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

function Copy-Directory {
    param(
        [string]$Source,
        [string]$Destination
    )

    if (-not (Test-Path -LiteralPath $Source -PathType Container)) {
        throw "Expected catalog directory missing: $Source"
    }

    New-Item -ItemType Directory -Force -Path $Destination | Out-Null
    Copy-Item -Path (Join-Path $Source "*") -Destination $Destination -Recurse -Force
}

function Get-RelativeFiles {
    param([string]$Root)

    if (-not (Test-Path -LiteralPath $Root -PathType Container)) {
        return @()
    }

    Get-ChildItem -LiteralPath $Root -File -Recurse | ForEach-Object {
        $_.FullName.Substring($Root.Length).TrimStart("\", "/")
    } | Sort-Object
}

Require-Command git

$ProjectRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$CatalogUrl = if ($env:NIAC_DEMO_CATALOG_URL) { $env:NIAC_DEMO_CATALOG_URL } else { "git@github.com:MustardSeedNetworks/niac-demo-catalog.git" }
$CatalogRef = if ($env:NIAC_DEMO_CATALOG_REF) { $env:NIAC_DEMO_CATALOG_REF } else { "main" }
$CatalogDir = if ($env:NIAC_DEMO_CATALOG_DIR) { $env:NIAC_DEMO_CATALOG_DIR } else { Join-Path $ProjectRoot ".catalog/niac-demo-catalog" }
$ExamplesDir = if ($env:NIAC_GO_EXAMPLES_DIR) { $env:NIAC_GO_EXAMPLES_DIR } else { Join-Path $ProjectRoot "examples" }
$Offline = $env:NIAC_DEMO_CATALOG_OFFLINE -eq "1"

if ($Offline) {
    if (-not (Test-Path -LiteralPath $CatalogDir -PathType Container)) {
        throw "NIAC_DEMO_CATALOG_OFFLINE=1 but catalog directory does not exist: $CatalogDir"
    }
} elseif (Test-Path -LiteralPath (Join-Path $CatalogDir ".git") -PathType Container) {
    git -C $CatalogDir fetch --depth 1 origin $CatalogRef
    git -C $CatalogDir checkout --detach FETCH_HEAD
} else {
    New-Item -ItemType Directory -Force -Path (Split-Path -Parent $CatalogDir) | Out-Null
    git clone --depth 1 --branch $CatalogRef $CatalogUrl $CatalogDir
}

$Stage = Join-Path ([System.IO.Path]::GetTempPath()) ("niac-go-demo-catalog-" + [System.Guid]::NewGuid())
New-Item -ItemType Directory -Force -Path $Stage | Out-Null

try {
    Copy-Directory (Join-Path $CatalogDir "scenarios/go-yaml") $Stage
    Copy-Directory (Join-Path $CatalogDir "walks/raw") (Join-Path $Stage "device_walks")
    Copy-Directory (Join-Path $CatalogDir "walks/sanitized") (Join-Path $Stage "device_walks_sanitized")
    Copy-Directory (Join-Path $CatalogDir "captures/shared") (Join-Path $Stage "captures")
    Copy-Directory (Join-Path $CatalogDir "captures/go-extra") (Join-Path $Stage "pcaps")
    Copy-Directory (Join-Path $CatalogDir "tools/walk-scripts/go") (Join-Path $Stage "walk_scripts")
    $SharedRunDemo = Join-Path $CatalogDir "tools/walk-scripts/java/run_demo.sh"
    if (Test-Path -LiteralPath $SharedRunDemo -PathType Leaf) {
        Copy-Item -LiteralPath $SharedRunDemo -Destination (Join-Path $Stage "walk_scripts/run_demo.sh") -Force
    }
    Copy-Directory (Join-Path $CatalogDir "docs/imported/go-examples") $Stage

    if ($Mode -eq "Sync") {
        if (Test-Path -LiteralPath $ExamplesDir) {
            Remove-Item -LiteralPath $ExamplesDir -Recurse -Force
        }
        New-Item -ItemType Directory -Force -Path $ExamplesDir | Out-Null
        Copy-Item -Path (Join-Path $Stage "*") -Destination $ExamplesDir -Recurse -Force
        Write-Host "OK: generated $ExamplesDir from the shared demo catalog."
    } else {
        $stageFiles = Get-RelativeFiles $Stage
        $exampleFiles = Get-RelativeFiles $ExamplesDir
        if (Compare-Object $stageFiles $exampleFiles) {
            throw "$ExamplesDir does not match the shared demo catalog. Run scripts/sync-demo-catalog.ps1 -Mode Sync."
        }
        foreach ($relative in $stageFiles) {
            $stageHash = (Get-FileHash -Algorithm SHA256 -LiteralPath (Join-Path $Stage $relative)).Hash
            $exampleHash = (Get-FileHash -Algorithm SHA256 -LiteralPath (Join-Path $ExamplesDir $relative)).Hash
            if ($stageHash -ne $exampleHash) {
                throw "File differs from shared demo catalog: $relative"
            }
        }
        Write-Host "OK: $ExamplesDir matches the shared demo catalog."
    }
} finally {
    Remove-Item -LiteralPath $Stage -Recurse -Force -ErrorAction SilentlyContinue
}
