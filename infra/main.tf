# PREPROD (TP8): un entorno que no existe hasta que esto se aplica.
# QA y PROD se crearon a mano en el TP6 y este codigo NO los toca: no aparecen
# en ningun plan porque no estan declarados.
#
# Uso (desde infra/, con NEON_API_KEY, RENDER_API_KEY y RENDER_OWNER_ID
# exportadas):  terraform init && terraform plan && terraform apply
# Y al terminar la sesion: terraform destroy (las 750 h de Render son del
# workspace entero: si se agotan, se suspende tambien PROD).

terraform {
  required_version = ">= 1.5"

  # Versiones FIJAS: un rango dejaria que init baje otra cosa el dia de la
  # defensa. El .terraform.lock.hcl fija ademas el hash de cada una.
  required_providers {
    neon = {
      source  = "kislerdm/neon"
      version = "0.18.0"
    }
    render = {
      source  = "render-oss/render"
      version = "1.9.1"
    }
    random = {
      source  = "hashicorp/random"
      version = "3.7.2"
    }
  }
}

# Sin credenciales aca: cada provider lee las suyas del entorno.
provider "neon" {}   # NEON_API_KEY
provider "render" {} # RENDER_API_KEY y RENDER_OWNER_ID

# --- La base: un proyecto de Neon propio, que se destruye entero sin rozar QA/PROD ---

resource "neon_project" "preprod" {
  name      = "tienda-preprod"
  region_id = var.region_neon

  # El default del provider (86400 s) supera el tope del plan gratuito (21600):
  # sin esta linea el primer apply falla.
  history_retention_seconds = 21600
}

resource "neon_role" "preprod" {
  project_id = neon_project.preprod.id
  branch_id  = neon_project.preprod.default_branch_id
  name       = "tienda_preprod_owner"
}

resource "neon_database" "preprod" {
  project_id = neon_project.preprod.id
  branch_id  = neon_project.preprod.default_branch_id
  name       = "tienda_preprod"
  owner_name = neon_role.preprod.name
}

# El secreto de los JWT lo genera Terraform: nadie lo tipea, no esta en ningun
# archivo del repo y cambia cada vez que preprod renace (los tokens viejos dejan
# de valer, que en un entorno desechable es lo esperable). Queda en el estado.
resource "random_password" "jwt" {
  length  = 64
  special = false
}

# --- Los servicios: la imagen del TP7, nunca un build ---

resource "render_web_service" "api" {
  name   = "tienda-api-preprod"
  plan   = "free"
  region = var.region_render

  runtime_source = {
    image = {
      image_url = var.image_repo_api # sin etiqueta: va en tag
      tag       = var.image_tag
    }
  }

  # La conexion se DERIVA de lo que devolvio Neon: nadie la copia. Es el error
  # mas caro del TP6 (un entorno contra la base de otro), cerrado por diseño.
  # Va por partes porque config.go la arma asi (DB_HOST no acepta una URL).
  # APP_COMMIT NO va aca: viene horneado en la imagen y una variable del
  # servicio lo pisaria, y /healthz dejaria de decir que imagen corre.
  env_vars = {
    DB_HOST    = { value = neon_project.preprod.database_host }
    DB_USER    = { value = neon_role.preprod.name }
    DB_PASS    = { value = neon_role.preprod.password }
    DB_SCHEMA  = { value = neon_database.preprod.name }
    DB_SSLMODE = { value = "require" }
    JWT_SECRET = { value = random_password.jwt.result }
  }
}

resource "render_web_service" "front" {
  name   = "tienda-front-preprod"
  plan   = "free"
  region = var.region_render

  runtime_source = {
    image = {
      image_url = var.image_repo_front
      tag       = var.image_tag
    }
  }

  # La direccion del backend sale del atributo url de SU api, no de un
  # copy-paste. trimsuffix por las dudas: con barra final nginx reescribe el
  # prefijo y cada /api/... da 404 (ver CLAUDE.md, proxy_pass sin barra).
  env_vars = {
    BACKEND_URL  = { value = trimsuffix(render_web_service.api.url, "/") }
    DNS_RESOLVER = { value = "8.8.8.8" }
  }
}
