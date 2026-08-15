$env:MYPAAS_AUDIT_BASE_URL="https://malala.tech"

# Run 1: Default (MyPaas - Compose app+db, GHCR registry, Invalid)
Write-Host "Running default scenarios..."
pnpm audit:create:prod
Copy-Item -Path "artifacts\create-project-audit" -Destination "artifacts\audit-default" -Recurse -Force

# Run 2: Subdir (Next.js with-docker)
Write-Host "Running subdir scenario..."
$env:MYPAAS_AUDIT_SUBDIR_REPO_URL="https://github.com/vercel/next.js"
$env:MYPAAS_AUDIT_SUBDIR_PATH="examples/with-docker"
pnpm audit:create:prod
Copy-Item -Path "artifacts\create-project-audit" -Destination "artifacts\audit-subdir" -Recurse -Force
Remove-Item Env:\MYPAAS_AUDIT_SUBDIR_REPO_URL
Remove-Item Env:\MYPAAS_AUDIT_SUBDIR_PATH

# Run 3: Static (Beginner HTML site)
Write-Host "Running static scenario..."
$env:MYPAAS_AUDIT_REPO_URL="https://github.com/mdn/beginner-html-site-styled"
pnpm audit:create:prod
Copy-Item -Path "artifacts\create-project-audit" -Destination "artifacts\audit-static" -Recurse -Force
Remove-Item Env:\MYPAAS_AUDIT_REPO_URL

# Run 4: Dockerfile (Getting started app)
Write-Host "Running dockerfile scenario..."
$env:MYPAAS_AUDIT_REPO_URL="https://github.com/docker/getting-started-app"
pnpm audit:create:prod
Copy-Item -Path "artifacts\create-project-audit" -Destination "artifacts\audit-dockerfile" -Recurse -Force
Remove-Item Env:\MYPAAS_AUDIT_REPO_URL

Write-Host "All audits complete!"
