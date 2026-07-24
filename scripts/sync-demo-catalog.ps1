param(
    [ValidateSet("Sync", "Check")]
    [string]$Mode = "Sync"
)

$ErrorActionPreference = "Stop"

function Require-Command {
    param([string]$Name)
    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "Missing required command: $Name"
    }
}

function Test-GitCheckout {
    param([string]$Path)
    git -C $Path rev-parse --is-inside-work-tree 2>$null | Out-Null
    return $LASTEXITCODE -eq 0
}

Require-Command git
Require-Command go

$ProjectRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$CatalogUrl = if ($env:NIAC_DEMO_CATALOG_URL) { $env:NIAC_DEMO_CATALOG_URL } else { "git@github.com:MustardSeedNetworks/niac-demo-catalog.git" }
$CatalogRef = if ($env:NIAC_DEMO_CATALOG_REF) { $env:NIAC_DEMO_CATALOG_REF } else { "main" }
$CatalogDir = if ($env:NIAC_DEMO_CATALOG_DIR) { $env:NIAC_DEMO_CATALOG_DIR } else { Join-Path $ProjectRoot ".catalog/niac-demo-catalog" }
$ExamplesDir = if ($env:NIAC_GO_EXAMPLES_DIR) { $env:NIAC_GO_EXAMPLES_DIR } else { Join-Path $ProjectRoot "examples" }
$Offline = $env:NIAC_DEMO_CATALOG_OFFLINE -eq "1"

if ($Offline) {
    if (-not (Test-GitCheckout $CatalogDir)) {
        throw "Offline catalog must be a git checkout: $CatalogDir"
    }
} elseif (Test-GitCheckout $CatalogDir) {
    git -C $CatalogDir fetch --depth 1 origin $CatalogRef
    if ($LASTEXITCODE -ne 0) { throw "Catalog fetch failed." }
} else {
    New-Item -ItemType Directory -Force -Path (Split-Path -Parent $CatalogDir) | Out-Null
    git clone --filter=blob:none --no-checkout $CatalogUrl $CatalogDir
    if ($LASTEXITCODE -ne 0) { throw "Catalog clone failed." }
    git -C $CatalogDir fetch --depth 1 origin $CatalogRef
    if ($LASTEXITCODE -ne 0) { throw "Catalog fetch failed." }
}

if (-not $Offline) {
    git -C $CatalogDir checkout --detach FETCH_HEAD
    if ($LASTEXITCODE -ne 0) { throw "Catalog checkout failed." }
}

$Dirty = git -C $CatalogDir status --porcelain --untracked-files=normal
if ($LASTEXITCODE -ne 0) { throw "Could not inspect the catalog checkout." }
if ($Dirty) { throw "Catalog checkout has uncommitted content: $CatalogDir" }

$SourceCommit = git -C $CatalogDir rev-parse HEAD
if ($LASTEXITCODE -ne 0) { throw "Could not resolve the catalog commit." }
$SourceUrl = git -C $CatalogDir remote get-url origin
if ($LASTEXITCODE -ne 0) { throw "Could not resolve the catalog origin URL." }
$ModeValue = $Mode.ToLowerInvariant()

Push-Location $ProjectRoot
try {
    go run ./cmd/niac-catalog-sync `
        -mode $ModeValue `
        -catalog-dir $CatalogDir `
        -examples-dir $ExamplesDir `
        -repository $SourceUrl `
        -commit $SourceCommit
    if ($LASTEXITCODE -ne 0) { throw "Catalog $Mode failed." }
} finally {
    Pop-Location
}
