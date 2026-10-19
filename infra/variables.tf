# Nada de esto es secreto (imagenes publicas, una etiqueta, regiones): por eso
# va con default en un archivo que se commitea.

variable "image_repo_api" {
  description = "Repositorio de la imagen del backend, SIN etiqueta"
  default     = "ghcr.io/kindguyj/tienda-backend"
}

variable "image_repo_front" {
  description = "Repositorio de la imagen del frontend, SIN etiqueta"
  default     = "ghcr.io/kindguyj/tienda-frontend"
}

# Preprod corre la imagen de un merge concreto, escrita aca: hoy la de v7.0.0.
# Cambiarla es un PR. Y ojo: en el plan gratuito de Render el cambio NO se
# aplica en caliente (el update falla); se pide la recreacion con
# terraform apply -replace=... o destroy + apply.
variable "image_tag" {
  description = "Etiqueta sha-<commit> publicada por el pipeline del TP7"
  default     = "sha-3b5697b204c6f1348ec7dd7f875ca88f8dab173a"
}

variable "region_render" {
  description = "frankfurt | ohio | oregon | singapore | virginia"
  default     = "oregon"
}

# La base cerca de los servicios: oregon es us-west-2.
variable "region_neon" {
  default = "aws-us-west-2"
}

# Sin default, igual que RENDER_OWNER_ID: es de QUE cuenta, no del entorno.
# Llega por TF_VAR_neon_org_id. Neon lo exige desde que las cuentas son
# organizaciones; sin el, crear el proyecto da 400 "org_id is required".
variable "neon_org_id" {
  type = string
}
