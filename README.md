# GoCloud CLI

Una herramienta de línea de comandos para generar y gestionar la estructura de proyectos de infraestructura como código usando Terraform/Terragrunt.

## ¿Qué hace GoCloud CLI?

GoCloud CLI te ayuda a:
- ✅ **Generar estructura de directorios** para proyectos Terraform/Terragrunt
- ✅ **Crear archivos de configuración** (main.tf, metadata.tf, terragrunt.hcl, etc.)
- ✅ **Gestionar secretos** en AWS SSM Parameter Store o SOPS
- ✅ **Configurar AWS SSO** automáticamente
- ✅ **Validar configuraciones** YAML antes de generar

**Importante**: GoCloud CLI **NO ejecuta terraform** - Solo genera archivos y estructura.

## Instalación

### Opción 1: Descargar binario (Recomendado)
Los binarios se publican en [GitHub Releases](https://github.com/gocloudLa/gocloud-cli/releases) en cada release. Nombres de artefactos: `gocloud-<versión>-<os>-<arch>` (ej. `gocloud-v1.2.3-darwin-arm64` para Apple Silicon).

```bash
# Ejemplo: descargar última versión para macOS ARM64
# Reemplaza VERSION por la versión deseada (ej. v1.2.3)
curl -sL -o gocloud "https://github.com/gocloudLa/gocloud-cli/releases/download/VERSION/gocloud-VERSION-darwin-arm64"
chmod +x gocloud
sudo mv gocloud /usr/local/bin/
```

### Opción 2: Compilar desde código
```bash
git clone https://github.com/gocloudLa/gocloud-cli.git
cd gocloud-cli
make build
sudo cp bin/gocloud /usr/local/bin/
```

## Inicio Rápido

```bash
# 1. Crear configuración de proyecto
gocloud config init my-project

# 2. Navegar al proyecto
cd my-project

# 3. Validar configuración
gocloud config validate

# 4. Generar estructura
gocloud generate

# 5. Configurar AWS SSO
gocloud sso setup

# 6. Hacer login a AWS
gocloud sso login --all

# 7. (Opcional) Inicializar secretos
gocloud secrets init --all
```

## Comandos Principales

Para ver la versión del CLI use `gocloud --version` o `gocloud version`.

### Versión y actualización

#### `gocloud version [--check] [--update]`
Muestra la versión actual del CLI.

**Opciones**:
- Sin flags: muestra la versión actual (build time y commit).
- `--check`: consulta [GitHub Releases](https://github.com/gocloudLa/gocloud-cli/releases) y compara con la última versión. Si hay una versión más nueva, imprime los comandos para actualizar manualmente (curl, chmod, mv).
- `--update`: si hay actualización disponible, intenta descargar y reemplazar el binario automáticamente (solo Unix; en Windows se muestran solo los comandos manuales).

**Ejemplos**:
```bash
gocloud version
gocloud version --check
gocloud version --check --update
```

Si hay una versión nueva y no usás `--update`, la salida incluirá los comandos exactos para tu plataforma, por ejemplo:
```bash
curl -sL -o gocloud "https://github.com/gocloudLa/gocloud-cli/releases/download/v1.1.0/gocloud-v1.1.0-darwin-arm64"
chmod +x gocloud
sudo mv gocloud /usr/local/bin/
```

### Configuración

#### `gocloud config init [project-name]`
Crea configuración inicial interactiva para tu proyecto.

**Funcionalidades**:
- Prompts interactivos para configuración de proyecto
- Validación de entrada en tiempo real
- Ambientes predefinidos (Shared, Production, Development, Staging)
- Recomendaciones de proyectos y workloads por ambiente
- Generación de archivo `gocloud.yaml`
- `--output`, `-o`: ruta del archivo de salida (default: project-name/gocloud.yaml)

```bash
gocloud config init my-project
gocloud config init my-project --skip-environments
gocloud config init my-project --skip-aws-sso
```

#### `gocloud config validate [--config file.yaml]`
Valida tu archivo de configuración antes de generar.

**Funcionalidades**:
- Validación completa del modelo de datos
- Detección de campos desconocidos/extra
- Modo estricto para validaciones adicionales
- Agregación de errores y warnings

```bash
gocloud config validate                    # Usa gocloud.yaml por defecto
gocloud config validate --config custom.yaml
gocloud config validate --strict
```

### Generación

#### `gocloud generate [--config file.yaml]`
Genera la estructura de directorios desde tu configuración.

**Funcionalidades**:
- Validación automática antes de generar
- Modo `--dry-run` para preview
- Confirmación granular por archivo (solo cuando el contenido cambia)
- Generación completa de estructura Terragrunt/Terraform
- Control granular de layers
- Generación de documentación (README.md)

```bash
gocloud generate                           # Usa gocloud.yaml por defecto
gocloud generate --dry-run                 # Preview sin generar archivos
gocloud generate --force                   # Sobrescribir sin confirmación
gocloud generate --working-dir custom-dir  # Directorio personalizado
```

**Comportamiento de confirmación**:
- **Nuevos directorios**: Se crean automáticamente sin preguntar
- **Archivos nuevos**: Se crean automáticamente sin preguntar  
- **Archivos `main.tf` existentes**: **NUNCA se sobrescriben** (protegidos)
- **Otros archivos existentes con contenido diferente**: Pregunta antes de sobrescribir
- **Archivos existentes con mismo contenido**: No pregunta, no sobrescribe
- **Flag `--force`**: Sobrescribe todos los archivos excepto `main.tf`

### AWS SSO

#### `gocloud sso setup`
Configura AWS SSO generando `.aws/config` desde tu configuración.

**Funcionalidades**:
- Genera archivo `.aws/config` con perfiles AWS SSO
- Crea directorio `.aws` con `.gitignore`
- Genera nombres de perfiles automáticamente: `{client}-{environment}`

```bash
gocloud sso setup                    # Usa gocloud.yaml por defecto
gocloud sso setup --config custom.yaml
```

#### `gocloud sso list`
Lista perfiles AWS disponibles.

**Funcionalidades**:
- Muestra perfiles numerados para fácil selección
- Valida que el archivo de configuración exista

```bash
gocloud sso list
```

#### `gocloud sso login`
Login interactivo a perfiles AWS SSO.

**Funcionalidades**:
- Modo interactivo con selección de perfiles
- Login a todos los perfiles con `--all`
- Login a perfiles específicos con `--profiles`
- Login en paralelo para múltiples perfiles

```bash
gocloud sso login                    # Modo interactivo
gocloud sso login --all              # Login a todos los perfiles
gocloud sso login --profiles prd,sha # Perfiles específicos
```

#### `gocloud sso verify`
Verifica el estado de tus perfiles AWS SSO.

**Funcionalidades**:
- Verificación automática de credenciales
- Comparación de account ID actual vs esperado
- Detección de credenciales expiradas o inválidas
- Verificación de todos los perfiles automáticamente

**Estados posibles**:
- ✅ **OK**: Credenciales válidas y account ID correcto
- ❌ **Expired/Invalid**: Credenciales expiradas o inválidas
- ❌ **Account Mismatch**: Account ID no coincide con el esperado

```bash
gocloud sso verify
```

### Gestión de Secretos

Gestiona secretos almacenados en **AWS SSM Parameter Store o en archivos SOPS** (YAML cifrado con AWS KMS). El backend (SSM vs SOPS) se elige en la configuración del proyecto.

**Funcionalidades generales**:
- Verificar existencia de secretos
- Listar, obtener, establecer y eliminar secretos individuales
- Editor de texto directo para editar secretos
- Validación JSON automática

#### `gocloud secrets check [layer-path]`
Verifica si los secretos existen según el backend configurado: con **SSM** verifica parámetros en AWS SSM Parameter Store; con **SOPS** verifica existencia y estado de `_secrets.yaml` (y KMS si aplica).

**Funcionalidades**:
- Verificación de existencia de secretos (SSM: parámetros; SOPS: archivo `_secrets.yaml`)
- Verificación por capa específica o por ambiente
- Verificación global de todas las capas

**Ejemplos**:
```bash
gocloud secrets check base/production    # Capa específica
gocloud secrets check --environment dev  # Todas las capas de un ambiente
gocloud secrets check --all              # Todas las capas
```

**Comportamiento**:
- Retorna estado de existencia de cada secreto
- Muestra errores claros si los secretos no existen
- Integra con perfiles AWS SSO configurados

#### `gocloud secrets init [layer-path]`
Inicializa secretos según el backend configurado: con **SSM** crea parámetros con JSON vacío en AWS SSM; con **SOPS** crea o asegura el archivo `_secrets.yaml` (YAML cifrado) y opcionalmente la KMS key por ambiente.

**Funcionalidades**:
- SSM: creación de parámetros con JSON vacío; SOPS: creación/aseguración de `_secrets.yaml` y KMS por ambiente
- Inicialización por capa específica o por ambiente
- Inicialización global de todas las capas

**Ejemplos**:
```bash
gocloud secrets init foundation/dev      # Capa específica
gocloud secrets init --environment dev   # Todas las capas de un ambiente
gocloud secrets init --all               # Todas las capas
```

**Comportamiento**:
- SSM: crea parámetros con estructura JSON vacía; SOPS: crea/asegura `_secrets.yaml` cifrado y KMS key si no existe
- No sobrescribe secretos existentes
- Integra con perfiles AWS SSO configurados

#### `gocloud secrets list <layer-path>`
Lista todos los secretos de una capa.

**Funcionalidades**:
- Listado de todos los secretos de una capa específica
- Formato legible de claves y valores
- Integración con AWS SSM Parameter Store

**Ejemplos**:
```bash
gocloud secrets list base/production
gocloud secrets list project/core/production
```

**Comportamiento**:
- Muestra todas las claves de secretos de la capa
- Formato JSON legible para valores
- Integra con perfiles AWS SSO configurados

#### `gocloud secrets get <layer-path> <key>`
Obtiene el valor de un secreto específico.

**Funcionalidades**:
- Obtención de valor de secreto específico
- Formato legible de salida
- Integración con AWS SSM Parameter Store

**Ejemplos**:
```bash
gocloud secrets get base/production database_url
gocloud secrets get project/core/production api_key
```

**Comportamiento**:
- Retorna el valor del secreto solicitado
- Muestra errores claros si el secreto no existe
- Integra con perfiles AWS SSO configurados

#### `gocloud secrets set <layer-path> <key> <value>`
Establece el valor de un secreto.

**Funcionalidades**:
- Establecimiento de valor de secreto específico
- Validación de formato JSON
- Integración con AWS SSM Parameter Store

**Ejemplos**:
```bash
gocloud secrets set base/production database_url "postgresql://..."
gocloud secrets set project/core/production api_key "secret-key"
```

**Comportamiento**:
- Actualiza o crea el secreto en AWS SSM
- Valida formato JSON automáticamente
- Integra con perfiles AWS SSO configurados

#### `gocloud secrets delete <layer-path> <key>`
Elimina un secreto específico de una capa.

**Funcionalidades**:
- Eliminación de una clave de secreto en una capa
- Integración con AWS SSM Parameter Store o SOPS (según backend configurado)

**Ejemplos**:
```bash
gocloud secrets delete base/production database_url
gocloud secrets delete project/core/production api_key
```

**Comportamiento**:
- Elimina la clave indicada del almacén de secretos de la capa
- Muestra errores claros si el secreto o la clave no existen
- Integra con perfiles AWS SSO configurados (backend SSM)

#### `gocloud secrets edit <layer-path>`
Editor de texto directo para editar secretos de una capa.

**Funcionalidades**:
- Editor de texto directo para editar secretos
- Validación JSON automática
- Preservación de cambios del usuario durante edición

```bash
gocloud secrets edit base/production
gocloud secrets edit project/core/production
```

### Módulos

Gestiona módulos de Terraform. Actualmente están disponibles los comandos de generación de documentación.

#### `gocloud module readme generate`
Genera documentación de módulos desde un template y un archivo YAML de configuración.

**Opciones**:
- `--yaml`, `-y`: Archivo YAML de entrada (default: `README.yml`)
- `--output`, `-o`: Archivo README de salida (default: `README.md`)
- `--template-url`: URL del template remoto (default: template embebido GoCloud)
- `--template-file`, `-t`: Archivo de template local (tiene prioridad sobre `--template-url`)
- `--terraform-dir`: Directorio Terraform para detección de módulos (default: `examples/complete`)

**Ejemplos**:
```bash
gocloud module readme generate
gocloud module readme generate --yaml custom.yml --output CUSTOM.md
gocloud module readme generate --template-file ./local-template.md.gotmpl
gocloud module readme generate --terraform-dir examples/my-module
```

#### `gocloud module readme generate-example`
Genera documentación de ejemplos de módulos desde un template y un archivo YAML.

**Opciones**:
- `--yaml`, `-y`: Archivo YAML de entrada (default: `README.yml`)
- `--output`, `-o`: Archivo README de salida (default: `README.md`)
- `--template-url`: URL del template remoto (default: template embebido GoCloud para ejemplos)
- `--template-file`, `-t`: Archivo de template local (tiene prioridad sobre `--template-url`)

**Ejemplos**:
```bash
gocloud module readme generate-example
gocloud module readme generate-example --yaml examples/complete/README.yml --output examples/complete/README-new.md
gocloud module readme generate-example --template-file ./local-example-template.md.gotmpl
```

### Autocompletado

#### `gocloud completion <shell>`
Genera scripts de autocompletado para diferentes shells.

**Funcionalidades**:
- Soporte para Bash, Zsh, Fish, PowerShell
- Autocompletado dinámico para `gocloud secrets`
- Autocompletado de layer paths desde configuración
- Autocompletado de secret keys (desde AWS SSM o desde archivos SOPS según el backend configurado)

```bash
gocloud completion bash
gocloud completion zsh
gocloud completion fish
gocloud completion powershell
```

## Archivo de Configuración `gocloud.yaml`

GoCloud CLI usa un archivo YAML para configurar tu proyecto. Este archivo define toda la estructura de infraestructura, ambientes, proyectos y workloads.

### Estructura General

```yaml
cli:
  # working_dir: "." (default)
  # auto_backup: true (default)
  # backup_dir: ".gocloud-backups" (default)
  # verbose: false (default)
  # debug: false (default)

infrastructure:
  client: "my-client"        # Nombre del cliente (requerido)
  company: "gcl"             # Prefijo de la empresa (requerido, 2-10 caracteres)
  region: "us-east-1"        # Región AWS por defecto (requerido)
  # version: "0.17.0" (default)
  # enable_secrets: true (default)
  # enable_terragrunt: true (default)
  
  # Backend de Terraform (opcional)
  backend:
    # pattern: "s3-backend" (default)
    # region: (usa infrastructure.region por defecto)
    # account: "sha" (default)
    # encrypt: true (default)
    bucket_name: "custom-bucket"        # Nombre personalizado (opcional)
    dynamodb_table_name: "custom-table" # Tabla personalizada (opcional)
  
  # AWS SSO (opcional)
  aws_sso:
    # region: (usa infrastructure.region por defecto)
    start_url: "https://my-client.awsapps.com/start#/"  # URL de inicio (requerido)
    role_name: "Admin"       # Nombre del rol (requerido)
  
  # Control de capas (opcional)
  layers:
    base: false              # Deshabilitar capa base
    # foundation: true (default)
    # organization: true (default)
  
  # Metadatos personalizados (opcional)
  metadata:
    public_domain: "gocloud.la"
    private_domain: "gocloud.private"
    internal_domain: "gocloud.internal"
  
  # Ambientes (requerido)
  environments:
    shared:
      name: "Shared"         # Nombre del ambiente (opcional)
      aws_account: "111111111111"  # Account ID (requerido)
      # enable_secrets: true (default: hereda del global)
      # enable_sso: true (default: hereda del global)
      # region: (default: usa infrastructure.region)
      # version: (default: usa infrastructure.version)
      projects: ["core", "common"]
      workloads: ["webapp", "api"]
```

### Configuración de Ambientes

Cada ambiente puede tener configuración específica. **Por defecto, los ambientes heredan la configuración global y usan la clave como nombre de directorio**.

#### Defaults
```yaml
# Por defecto (no necesitas especificar esto):
infrastructure:
  environments:
    dev:
      aws_account: "111111111111"
      # name: (usa la clave "dev")
      # dir_name: (usa la clave "dev")
      # enable_secrets: true (hereda del global)
      # enable_sso: true (hereda del global)
      # region: (usa infrastructure.region)
      # version: (usa infrastructure.version)
```

#### Configuración Personalizada

```yaml
infrastructure:
  environments:
    # Ambiente simple
    dev:
      name: "Development"
      aws_account: "111111111111"
      projects: ["core", "common"]
      workloads: ["webapp", "api"]
    
    # Ambiente con directorio personalizado
    shared:
      name: "Shared"
      dir_name: "shared"
      aws_account: "111111111111"
      projects: ["core", "common"]
      workloads: ["webapp", "api"]
    
    # Ambiente con configuración avanzada
    production:
      name: "Production"
      dir_name: "production"
      aws_account: "222222222222"
      region: "eu-west-1"           # Región específica
      version: "v2.14.0"            # Versión específica
      enable_secrets: false         # Deshabilitar secretos
      enable_sso: false             # Deshabilitar SSO
      
      # Control de capas por ambiente
      layers:
        base: false                 # Deshabilitar base
        # foundation: true (hereda del global)
        # organization no se puede configurar por ambiente
      
      # SSO específico del ambiente
      aws_sso:
        start_url: "https://prod.awsapps.com/start#/"
        role_name: "ProductionAdmin"
      
      # Proyectos y workloads
      projects: ["core", "common"]
      workloads: ["webapp", "api"]
```

**Comportamiento**:
- **Por defecto**: Los ambientes heredan configuración global y usan la clave como nombre de directorio
- **Personalizado**: Puedes sobrescribir cualquier configuración por ambiente
- **Prioridad**: Ambiente específico > Global
- **Nombres de directorio**: Por defecto usa la clave, con `dir_name` puedes personalizarlo

### Proyectos y Workloads

Puedes definir proyectos y workloads para cada ambiente. **Por defecto, se usan las claves como nombres de directorio**.

#### Defaults
```yaml
# Por defecto (usa las claves como nombres):
environments:
  production:
    projects: ["core", "common"]    # Directorios: "core", "common"
    workloads: ["webapp", "api"]    # Directorios: "webapp", "api"
```

#### Configuración Simple

```yaml
environments:
  production:
    projects: ["core", "common"]
    workloads: ["webapp", "api"]
```

#### Configuración Avanzada con Nombres Personalizados

```yaml
environments:
  production:
    projects:
      - core
      - common
      # Proyecto con nombre personalizado
      - dept:                    # Key del proyecto
          name: "Deposits"       # Nombre en metadata.tf
      # Proyecto con directorio personalizado
      - wdwl:                    # Key del proyecto
          name: "Withdrawals"    # Nombre en metadata.tf
          dir_name: "withdrawals" # Directorio personalizado
    
    workloads:
      - webapp
      - api
      # Workload con dependencias personalizadas
      - blockchain-service:
          depends_on: ["project/common", "project/core"]
      # Workload con secretos deshabilitados
      - legacy-app:
          enable_secrets: false
      # Workload con nombre y directorio personalizados
      - dept:                    # Key del workload
          name: "Deposits"       # Nombre en metadata.tf
          dir_name: "deposits"   # Directorio personalizado
          enable_secrets: false  # Sin archivos _secrets.tf
```

**Comportamiento**:
- **Por defecto**: Se usan las claves como nombres de directorio
- **Con `name`**: Se convierte a minúsculas para el directorio
- **Con `dir_name`**: Se usa el valor exacto especificado

### Control de Capas

Puedes controlar qué capas se generan a diferentes niveles. **Por defecto, todas las capas se generan**.

#### Defaults
```yaml
# Por defecto (no necesitas especificar esto):
infrastructure:
  layers:
    base: true           # Capa base (infraestructura compartida)
    foundation: true     # Capa foundation
    organization: true   # Configuración organizacional (genera main.tf y metadata.tf)
```

#### Control Global

Solo especifica `layers` si quieres **deshabilitar** alguna capa globalmente:

```yaml
infrastructure:
  layers:
    base: false          # No generar base en ningún ambiente
    # foundation y organization se generan (default)
```

#### Control por Ambiente

Solo especifica `layers` en un ambiente si quieres **deshabilitar** alguna capa específicamente:

```yaml
infrastructure:
  environments:
    staging:
      layers:
        base: false      # No generar base solo en staging
        # foundation se genera (hereda del global)
        # organization no se puede configurar por ambiente
```

**Comportamiento**:
- **Por defecto**: Todas las capas se generan en todos los ambientes
- **Global**: Solo se puede deshabilitar globalmente
- **Por ambiente**: Solo `base` y `foundation` se pueden deshabilitar por ambiente
- **`organization`**: Solo se genera si defines `infrastructure.organization.aws_account` (obligatorio para backend, secrets y SSO). Se deshabilita con `layers.organization: false`. Genera `main.tf`, `metadata.tf`, `backend.tf`, `providers.tf`, etc. con `env = "org"` y `environment = "Organization"`

### Control de Secretos

Puedes deshabilitar secretos a diferentes niveles con jerarquía de prioridad y elegir el backend de secretos (SSM o SOPS). **Por defecto, los secretos están habilitados y usan SSM**.

#### Defaults
```yaml
# Por defecto (no necesitas especificar esto):
# enable_secrets: true (definido en Estructura General)
# secrets: (no especificado = usa SSM por defecto)
```

#### Backend de Secretos (SSM o SOPS)

Puedes elegir entre dos backends de secretos:
- **SSM** (por defecto): Usa AWS Systems Manager Parameter Store
- **SOPS**: Usa archivos YAML cifrados con SOPS y AWS KMS

**Configuración jerárquica del backend** (misma jerarquía que `enable_secrets`):

**1. Global (menor prioridad)**
```yaml
infrastructure:
  secrets:
    type: "sops"  # Usar SOPS para todos los ambientes (default: "ssm")
```

**2. Por Ambiente**
```yaml
infrastructure:
  environments:
    production:
      secrets:
        type: "sops"  # Usar SOPS solo en production
```

**3. Por Project o Workload (mayor prioridad)**
```yaml
infrastructure:
  environments:
    production:
      projects:
        - example:
            secrets:
              type: "sops"  # Usar SOPS solo para este project
      workloads:
        - legacy-app:
            secrets:
              type: "sops"  # Usar SOPS solo para este workload
```

**Comportamiento del backend**:
- **Jerarquía**: Project/Workload > Environment > Global > Default ("ssm")
- **SSM**: Genera `_secrets.tf` con `data.aws_ssm_parameter` y almacena secretos en AWS SSM Parameter Store
- **SOPS**: Genera `_secrets.tf` con `data.sops_file` y almacena secretos en archivo `_secrets.yaml` cifrado con SOPS
- **KMS para SOPS**: Se crea automáticamente una KMS key por ambiente con alias `alias/{company}-{env}-secrets`
- **Archivos versionados**: Todos los archivos (`_secrets.yaml`, `_secrets.tf`) se versionan en git

#### Jerarquía de Prioridad para Habilitación

**1. Global (menor prioridad)**
```yaml
infrastructure:
  enable_secrets: false  # Deshabilitar secretos en todos los ambientes
```

**2. Por Ambiente**
```yaml
infrastructure:
  environments:
    staging:
      enable_secrets: false  # Deshabilitar solo en staging
```

**3. Por Workload (mayor prioridad)**
```yaml
infrastructure:
  environments:
    production:
      workloads:
        - legacy-app:
            enable_secrets: false  # Deshabilitar solo este workload
```

**Comportamiento**:
- **Por defecto**: `enable_secrets: true` (secretos habilitados)
- **Backend por defecto**: `secrets.type: "ssm"` (si no se especifica)
- **Generación**: Los archivos `_secrets.tf` no se generan si están deshabilitados
- **Comandos**: Los comandos `gocloud secrets` detectan automáticamente el backend según la configuración
- **Requisitos SOPS**: Requiere el binario `sops` instalado y permisos KMS adecuados

### Control de Terragrunt

Puedes deshabilitar la generación de archivos `terragrunt.hcl` a diferentes niveles con jerarquía de prioridad. **Por defecto, los archivos terragrunt.hcl están habilitados**.

#### Defaults
```yaml
# Por defecto (no necesitas especificar esto):
# enable_terragrunt: true (definido en Estructura General)
```

#### Jerarquía de Prioridad

**1. Global (menor prioridad)**
```yaml
infrastructure:
  enable_terragrunt: false  # Deshabilitar terragrunt en todos los ambientes
```

**2. Por Ambiente**
```yaml
infrastructure:
  environments:
    staging:
      enable_terragrunt: false  # Deshabilitar solo en staging
```

**3. Por Project (mayor prioridad)**
```yaml
infrastructure:
  environments:
    production:
      projects:
        - legacy-system:
            enable_terragrunt: false  # Deshabilitar solo este project
```

**4. Por Workload (mayor prioridad)**
```yaml
infrastructure:
  environments:
    production:
      workloads:
        - legacy-app:
            enable_terragrunt: false  # Deshabilitar solo este workload
```

**Comportamiento**:
- **Por defecto**: `enable_terragrunt: true` (archivos terragrunt.hcl habilitados)
- **Generación**: Los archivos `terragrunt.hcl` no se generan si están deshabilitados
- **Eliminación**: Si existe un archivo `terragrunt.hcl` y se deshabilita, se elimina automáticamente
- **Jerarquía**: Workload / Project > Environment > Global

### Control de SSO

Puedes deshabilitar la generación de perfiles SSO para ambientes específicos y personalizar la configuración SSO por ambiente. **Por defecto, todos los ambientes generan perfiles SSO usando la configuración global**.

#### Defaults
```yaml
# Por defecto (no necesitas especificar esto):
infrastructure:
  environments:
    dev:
      aws_account: "111111111111"
      # enable_sso: true (default)
      # aws_sso: (usa configuración global)
```

#### Deshabilitar SSO por Ambiente

```yaml
infrastructure:
  environments:
    dev:
      aws_account: "111111111111"
      # enable_sso: true (default)
    
    staging:
      aws_account: "222222222222"
      enable_sso: false  # No generar perfil SSO para staging
      layers:
        base: false      # Típicamente usado con layers deshabilitados
        foundation: false
```

#### SSO Específico por Ambiente

```yaml
infrastructure:
  aws_sso:
    region: "us-east-1"
    start_url: "https://my-client.awsapps.com/start#/"
    role_name: "Admin"
  
  environments:
    shared:
      aws_account: "111111111111"
      aws_sso:
        start_url: "https://shared.awsapps.com/start#/"
        role_name: "SharedAdmin"
    
    production:
      aws_account: "222222222222"
      aws_sso:
        role_name: "ProductionAdmin"
        # start_url: (usa el global)
```

**Comportamiento**:
- **Por defecto**: `enable_sso: true` (todos los ambientes generan perfiles SSO)
- **Deshabilitado**: No se genera perfil SSO, no aparece en `aws_accounts`
- **Específico**: Puedes sobrescribir `start_url` y `role_name` por ambiente

### Regiones por Ambiente

Puedes especificar regiones diferentes para cada ambiente. **Por defecto, todos los ambientes usan la región global**.

#### Defaults
```yaml
# Por defecto (usa la región global):
# region: (definido en Estructura General)
environments:
  dev:
    aws_account: "111111111111"
    # region: (usa infrastructure.region)
```

#### Regiones Específicas

```yaml
infrastructure:
  environments:
    dev:
      # Sin región específica = usa la global
      aws_account: "111111111111"
    
    production:
      region: "eu-west-1"  # Región específica para producción
      aws_account: "222222222222"
```

**Comportamiento**:
- **Por defecto**: Todos los ambientes usan `infrastructure.region`
- **Específica**: Los archivos `metadata.tf` usan la región específica del ambiente

### Versiones por Ambiente

Puedes especificar versiones diferentes de módulos por ambiente. **Por defecto, todos los ambientes usan la versión global**.

#### Defaults
```yaml
# Por defecto:
# version: (definido en Estructura General)
environments:
  dev:
    aws_account: "111111111111"
    # Sin version específica = usa la global
```

#### Versiones Específicas por Ambiente

```yaml
infrastructure:
  environments:
    sha:
      version: "latest"   # Usa 'latest' para este ambiente
    stg:
      # Sin versión específica = usa la global
    prd:
      version: "v2.14.0"  # Usa una versión específica
```

#### Gestión Inteligente de Versiones

GoCloud CLI detecta automáticamente cambios de versión y actualiza solo los archivos necesarios:

```bash
# Cambiar versión global
infrastructure:
  version: "0.17.0"  # Actualiza todos los environments sin versión específica

# Cambiar versión por environment
environments:
  sha:
    version: "latest"  # Solo actualiza archivos del environment 'sha'
  prd:
    version: "v2.14.0"  # Solo actualiza archivos del environment 'prd'
```

**Logs de actualización**:
```
INFO: Version change detected: v2.14.0 -> latest
INFO: File 'base/shared/main.tf' will be updated due to version change
INFO: File 'foundation/shared/main.tf' will be updated due to version change
```

**Comportamiento**:
- **Por defecto**: Todos los ambientes usan `infrastructure.version`
- **Prioridad**: Ambiente específica > Global
- **Detección automática**: Compara versión actual vs configurada
- **Actualización selectiva**: Solo se actualizan archivos del ambiente afectado

### Source Personalizado

Puedes usar módulos desde Git en lugar del registry de Terraform. **Por defecto, se usa el registry oficial**.

#### Defaults
```yaml
# Por defecto (usa registry):
# source: (no especificado)
# source_ref: (no especificado)
```

#### Source Git Global

```yaml
infrastructure:
  # Usar Git source para todos los ambientes
  source: "git@github.com:gocloudLa/terraform-aws-standard-platform.git"
  source_ref: "main"  # branch, tag, o commit hash
```

#### Source Git por Ambiente

```yaml
infrastructure:
  source: "git@github.com:gocloudLa/terraform-aws-standard-platform.git"
  source_ref: "main"
  
  environments:
    dev:
      # Usar feature branch para desarrollo
      source: "git@github.com:gocloudLa/terraform-aws-standard-platform.git"
      source_ref: "feature/new-feature"
    
    stg:
      # Usar registry para staging
      # source: (no especificado = usa registry)
      version: "0.2.0"
    
    prd:
      # Usar tag específico para producción
      source: "git@github.com:gocloudLa/terraform-aws-standard-platform.git"
      source_ref: "v1.0.0"
```

**Resultado en main.tf**:

**Con Git source**:
```hcl
module "base" {
  source = "git@github.com:gocloudLa/terraform-aws-standard-platform.git//modules/base?ref=feature/new-feature"
  
  metadata = local.metadata
}
```

**Con Registry**:
```hcl
module "base" {
  source  = "gocloudLa/standard-platform/aws//modules/base"
  version = "0.17.0"
  
  metadata = local.metadata
}
```

**Comportamiento**:
- **Por defecto**: Se usa el registry de Terraform con `version`
- **Con Git**: Se usa `source` con `?ref=` y NO se incluye `version`
- **Prioridad**: Ambiente específico > Global > Registry (fallback)
- **Rollback fácil**: Eliminar o comentar `source` para volver al registry

### Metadatos Personalizados

Puedes definir metadatos personalizados que se insertan en todos los archivos `metadata.tf`. **Por defecto, no hay metadatos personalizados**.

#### Defaults
```yaml
# Por defecto (no hay metadatos personalizados):
infrastructure:
  # metadata: (no especificado)
```

#### Configuración de Metadatos

```yaml
infrastructure:
  metadata:
    public_domain: "gocloud.la"
    private_domain: "gocloud.private"
    internal_domain: "gocloud.internal"
    company_email: "devops@gocloud.la"
    support_team: "platform"
```

**Resultado en metadata.tf**:
```hcl
metadata = {
  aws_region  = "us-east-1"
  environment = "Development"
  
  public_domain   = "gocloud.la"
  private_domain  = "gocloud.private"
  internal_domain = "gocloud.internal"
  company_email   = "devops@gocloud.la"
  support_team    = "platform"
  
  key = {
    company = "gcl"
    region  = "use1"
    env     = "dev"
    layer   = "base"
  }
}
```

**Comportamiento**:
- **Por defecto**: No se agregan metadatos personalizados
- **Inserción automática**: Se insertan en todos los archivos `metadata.tf`

### Configuración de Backend y Providers

Puedes configurar el backend de Terraform y los providers. **GoCloud CLI genera automáticamente `providers.tf` y `backend.tf`** reemplazando la generación de Terragrunt. El campo `type` es el tipo de backend de Terraform (p. ej. "s3"); `pattern` es un nombre de patrón para construir nombres de bucket y tabla (p. ej. s3-backend, tf-backend).

#### Valores por Defecto

Si **NO defines** `backend:` ni `providers:`, GoCloud CLI usa estos valores por defecto:

**Backend defaults:**
```yaml
# backend:
#   pattern: "s3-backend"        # Default: "s3-backend"
#   region: "us-east-1"          # Default: "us-east-1" 
#   account: "sha"               # Default: "sha"
#   encrypt: true                # Default: true
#   type: "s3"                  # Default: "s3"
#   use_profile: true           # Default: true
```

**Providers defaults:**
```yaml
# providers:
#   default_providers:
#     - name: "aws"
#       region: "local.metadata.aws_region"
#     - name: "aws"
#       region: "us-east-1"
#       alias: "use1"
#   use_profiles: true          # Default: true
```

**Campos opcionales (se construyen automáticamente si no se definen):**
- **Bucket**: `{company}-{account}-{pattern}` (ej: `gcl-sha-s3-backend`)
- **DynamoDB**: `{company}-{account}-{pattern}` (ej: `gcl-sha-s3-backend`)

#### Configuración Básica (Solo Global)

```yaml
infrastructure:
  backend:
    pattern: "s3-backend"    # Patrón del backend
    region: "us-east-1"      # Región del backend
    account: "sha"          # Environment donde está el backend
    encrypt: true            # Encriptación del estado
    bucket_name: "mi-bucket-terraform"        # Opcional: nombre personalizado
    dynamodb_table_name: "mi-tabla-locks"     # Opcional: nombre personalizado
    role_template: "{{.Company}}-{{.BackendAccount}}-{{.AccountID}}"  # Template personalizado

  # Configuración de providers
  providers:
    default_providers:
      - name: "aws"
        region: "local.metadata.aws_region"
      - name: "aws"
        region: "us-east-1"
        alias: "use1"
    use_profiles: true  # Control de AWS profiles (default: true)
```

#### Configuración Jerárquica (Global + Environment + Project/Workload)

```yaml
infrastructure:
  backend:
    type: "s3"
    key_template: "{{.AccountID}}/{{.Layer}}/terraform.tfstate"
    role_template: "{{.Company}}-{{.BackendAccount}}-{{.AccountID}}"
    use_profile: true
  providers:
    use_profiles: true
    default_providers:
      - name: "aws"
        region: "local.metadata.aws_region"

environments:
  prd:
    backend:
      role_template: "custom-{{.BackendAccount}}-{{.AccountID}}"
      use_profile: false
    providers:
      use_profiles: false
```

#### Nuevas Features de Backend

```yaml
infrastructure:
  backend:
    # === NUEVAS FEATURES ===
    type: "s3"  # Tipo de backend (default: "s3")
    key_template: "{{.AccountID}}/{{.Layer}}-{{.Project}}-{{.Environment}}/terraform.tfstate"
    use_profile: true  # Control de AWS profiles (default: true)
    role_template: "{{.Company}}-{{.BackendAccount}}-{{.AccountID}}"  # Template personalizado para assume_role
```

**Variables disponibles en `key_template`**:
- `{{.AccountID}}` - AWS Account ID (ej: "123456789012")
- `{{.Layer}}` - Tipo de layer (ej: "base", "foundation", "project", "workload")
- `{{.Project}}` - Clave del project (solo para project/workload layers, ej: "core", "dept")
- `{{.Environment}}` - Clave del environment (ej: "prd", "dev", "stg")
- `{{.EnvironmentName}}` - Nombre del environment en minúsculas (ej: "production")
- `{{.Company}}` - Prefijo de la empresa (ej: "gcl")
- `{{.Region}}` - Región AWS (ej: "us-east-1")
- `{{.Client}}` - Nombre del cliente (ej: "test-client")

**Variables disponibles en `role_template`**:
- `{{.AccountID}}` - AWS Account ID del environment actual (ej: "123456789012")
- `{{.BackendAccountID}}` - AWS Account ID del environment del backend (ej: "123456789013")
- `{{.Company}}` - Prefijo de la empresa (ej: "gcl")
- `{{.BackendAccount}}` - Clave del environment del backend (ej: "sha")
- `{{.Layer}}` - Tipo de layer (ej: "base", "foundation", "project", "workload")
- `{{.Project}}` - Clave del project (solo para project/workload layers)
- `{{.Environment}}` - Clave del environment (ej: "prd", "dev", "stg")
- `{{.EnvironmentName}}` - Nombre del environment en minúsculas (ej: "production")
- `{{.Region}}` - Región AWS (ej: "us-east-1")
- `{{.Client}}` - Nombre del cliente (ej: "test-client")

#### Template Personalizado para Assume Role

Puedes personalizar el nombre del rol en `assume_role` usando `role_template`:

```yaml
infrastructure:
  backend:
    # Template personalizado para assume_role
    role_template: "{{.Company}}-{{.BackendAccount}}-{{.AccountID}}"
```

**Ejemplos de uso**:

```yaml
# Template por defecto (sin role_template)
# Genera: "gcl-sha-tf-backend-123456789012"
backend:
  pattern: "tf-backend"
  account: "sha"

# Template personalizado simple
# Genera: "gcl-sha-123456789012"
backend:
  role_template: "{{.Company}}-{{.BackendAccount}}-{{.AccountID}}"

# Template para casos específicos
# Genera: "bdt-inf-026090514459"
backend:
  role_template: "bdt-{{.BackendAccount}}-{{.AccountID}}"

# Template con patrón específico del cliente
# Genera: "custom-role-123456789012"
backend:
  role_template: "custom-role-{{.AccountID}}"
```

**Comportamiento**:
- **Sin `role_template`**: Usa el patrón por defecto `{{.Company}}-{{.BackendAccount}}-{{.BackendPattern}}-{{.AccountID}}`
- **Con `role_template`**: Usa el template personalizado especificado
- **Variables**: Todas las variables de template están disponibles para personalización
- **Fallback**: Si no se especifica `role_template`, se usa el patrón por defecto

#### Jerarquía de Configuración

```yaml
infrastructure:
  backend:
    type: "s3"
    key_template: "{{.AccountID}}/{{.Layer}}/terraform.tfstate"  # Global
    role_template: "{{.Company}}-{{.BackendAccount}}-{{.AccountID}}"  # Global
    use_profile: true
  providers:
    use_profiles: true  # Global
    default_providers:
      - name: "aws"
        region: "local.metadata.aws_region"

environments:
  prd:
    backend:
      key_template: "{{.Company}}/{{.Environment}}/{{.Layer}}/terraform.tfstate"  # Override environment
      role_template: "custom-{{.BackendAccount}}-{{.AccountID}}"  # Override environment
      use_profile: false  # Override environment
    providers:
      use_profiles: false  # Override environment
      default_providers:
        - name: "aws"
          region: "us-west-2"

    projects:
      - core:
          backend:
            key_template: "{{.Company}}/core/{{.Environment}}/terraform.tfstate"  # Override project
            role_template: "project-{{.BackendAccount}}-{{.AccountID}}"  # Override project
          providers:
            use_profiles: true  # Override project
            default_providers:
              - name: "aws"
                region: "us-east-1"
                alias: "primary"
    workloads:
      - api:
          backend:
            use_profile: true  # Override workload
            role_template: "workload-{{.BackendAccount}}-{{.AccountID}}"  # Override workload
          providers:
            use_profiles: true  # Override workload
            default_providers:
              - name: "aws"
                region: "eu-west-1"
                alias: "europe"
```

**Comportamiento**:
- **Por defecto**: Se construyen automáticamente como `{company}-{account}-{pattern}`
- **Con nombres personalizados**: Se usan los valores especificados en `bucket_name` y `dynamodb_table_name`
- **Jerarquía**: Workload > Project > Environment > Global
- **Profiles**: Se incluyen automáticamente cuando AWS SSO está configurado
- **Providers**: Configuración jerárquica con `default_providers` y `use_profiles`
- **Role Template**: Personalización completa del nombre del rol en `assume_role`
- **Generación**: `providers.tf` y `backend.tf` se generan automáticamente

### Dependencias de Workloads

Puedes personalizar las dependencias de workloads. **Por defecto, las dependencias se asignan automáticamente**.

#### Dependencias por Defecto

```yaml
# Por defecto (dependencias automáticas):
# - base: no depende de nada
# - foundation: depende de base/{env}
# - project/{name}: depende de foundation/{env}
# - workload/{name}: depende de project/{name}/{env} (si existe) o project/common/{env} (fallback)
```

#### Dependencias Personalizadas

```yaml
environments:
  production:
    workloads:
      - blockchain-service:
          depends_on: ["project/common", "project/core"]
      - standalone-app:
          depends_on: []  # Sin dependencias
```

**Comportamiento**:
- **Por defecto**: Dependencias automáticas basadas en la jerarquía de capas
- **Personalizadas**: Se pueden especificar rutas relativas
- **Sin dependencias**: Se pueden deshabilitar con `depends_on: []`

### Nombres de Directorios

Puedes personalizar los nombres de directorios para proyectos y workloads. **Por defecto, se usa la clave del proyecto/workload**.

#### Defaults
```yaml
# Por defecto (usa la clave):
projects:
  - core        # Directorio: "core"
  - common      # Directorio: "common"

workloads:
  - webapp      # Directorio: "webapp"
  - api         # Directorio: "api"
```

#### Jerarquía de Nombres

**Prioridad** (de mayor a menor):
1. **`dir_name`** - Directorio personalizado específico
2. **`name`** - Nombre personalizado convertido a minúsculas
3. **`key`** - Clave del proyecto/workload (fallback)

#### Ejemplos

```yaml
# Proyecto con nombre personalizado
- dept:                    # Key: "dept"
    name: "Deposits"       # Directorio: "deposits" (name en minúsculas)

# Proyecto con directorio personalizado
- wdwl:                    # Key: "wdwl"
    name: "Withdrawals"    # Nombre: "Withdrawals"
    dir_name: "withdrawals" # Directorio: "withdrawals"
```

**Comportamiento**:
- **Por defecto**: Se usa la clave del proyecto/workload como nombre de directorio
- **Con `name`**: Se convierte a minúsculas para el directorio
- **Con `dir_name`**: Se usa el valor exacto especificado

## Autocompletado

### macOS (Zsh - Recomendado)

```bash
# Agregar autocompletado a ~/.zshrc
echo 'source <(gocloud completion zsh)' >> ~/.zshrc

# Recargar configuración
source ~/.zshrc

# Probar autocompletado
gocloud <TAB><TAB>
```

### macOS (Bash)

```bash
# Instalar bash-completion
brew install bash-completion@2

# Agregar a ~/.bash_profile
echo '[[ -r "$(brew --prefix)/etc/profile.d/bash_completion.sh" ]] && . "$(brew --prefix)/etc/profile.d/bash_completion.sh"' >> ~/.bash_profile
echo 'source <(gocloud completion bash)' >> ~/.bash_profile
source ~/.bash_profile
```

### Linux

```bash
# Ubuntu/Debian
sudo apt install bash-completion

# CentOS/RHEL/Fedora
sudo yum install bash-completion
# o
sudo dnf install bash-completion

# Configurar autocompletado
echo 'source <(gocloud completion bash)' >> ~/.bashrc
source ~/.bashrc
```

## Casos de Uso

### 1. Proyecto Nuevo Completo

```bash
# Crear proyecto desde cero
gocloud config init my-new-project
cd my-new-project

# Validar y generar
gocloud config validate
gocloud generate

# Configurar AWS y secretos
gocloud sso setup
gocloud sso login --all
gocloud secrets init --all
```

### 2. Proyecto con Configuración Personalizada

```bash
# Usar configuración de ejemplo como base
cp example/gocloud.yaml my-config.yaml

# Editar configuración
nano my-config.yaml

# Generar proyecto
gocloud generate --config my-config.yaml --working-dir my-project
```

### 3. Solo Generar Estructura (Sin AWS)

```bash
# Crear configuración sin AWS SSO
gocloud config init my-project --skip-aws-sso

# Generar solo estructura
gocloud generate --dry-run  # Preview
gocloud generate            # Generar
```

### 4. Gestionar Secretos Existentes

```bash
# Verificar qué secretos existen
gocloud secrets check --all

# Inicializar secretos faltantes
gocloud secrets init --environment dev

# Editar secretos
gocloud secrets edit base/production
```

## Troubleshooting

### Error: "failed to get shared config profile, default"

```bash
# El problema es que no existe .aws/config
# Solución: Configurar AWS SSO primero

gocloud sso setup
gocloud sso login --all
```

### Autocompletado no funciona en macOS

```bash
# Si obtienes error "command not found: compdef"
# Agregar al INICIO de ~/.zshrc:

echo 'autoload -Uz compinit' | cat - ~/.zshrc > ~/.zshrc.tmp && mv ~/.zshrc.tmp ~/.zshrc
echo 'compinit' | cat - ~/.zshrc > ~/.zshrc.tmp && mv ~/.zshrc.tmp ~/.zshrc

# Recargar
source ~/.zshrc
```

### AWS CLI no encontrado

```bash
# macOS
brew install awscli

# Linux
curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
unzip awscliv2.zip
sudo ./aws/install
```

### Archivos main.tf no se actualizan

Los archivos `main.tf` están protegidos contra sobrescritura para preservar tu lógica personalizada. Solo se actualizan automáticamente cuando cambia la versión de los módulos.

## Development

### Prerequisites
- Go 1.25.1
- golangci-lint (for linting)

### Install Development Tools
```bash
# Linting
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Development Commands
```bash
make lint          # Run linter
make test          # Run tests
make build         # Build binary
make fmt           # Format code
make pre-commit    # Run pre-commit checks (fmt, lint, test)
make build-current # Build for current platform only
make clean         # Clean build artifacts
make test-coverage # Run tests with coverage report
make quality       # All quality checks
make deps-check    # Check for outdated dependencies
```
La lista completa de targets está en `make help`.

## Licencia

Este proyecto está bajo la Licencia MIT. Ver el archivo `LICENSE` para más detalles.
