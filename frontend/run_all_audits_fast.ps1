$env:MYPAAS_AUDIT_BASE_URL="https://malala.tech"

# Run 2: Subdir
Write-Host "Running subdir scenario..."
$env:MYPAAS_AUDIT_SUBDIR_REPO_URL="https://github.com/dockersamples/node-bulletin-board"
$env:MYPAAS_AUDIT_SUBDIR_PATH="bulletin-board-app"
pnpm audit:create:prod
Copy-Item -Path "artifacts\create-project-audit" -Destination "artifacts\audit-subdir" -Recurse -Force
Remove-Item Env:\MYPAAS_AUDIT_SUBDIR_REPO_URL
Remove-Item Env:\MYPAAS_AUDIT_SUBDIR_PATH

# Run 3: Static
Write-Host "Running static scenario..."
$env:MYPAAS_AUDIT_REPO_URL="https://github.com/mdn/beginner-html-site-styled"
pnpm audit:create:prod
Copy-Item -Path "artifacts\create-project-audit" -Destination "artifacts\audit-static" -Recurse -Force
Remove-Item Env:\MYPAAS_AUDIT_REPO_URL

# Run 4: Dockerfile
Write-Host "Running dockerfile scenario..."
$env:MYPAAS_AUDIT_REPO_URL="https://github.com/nabilrn/mypaas-statd"
pnpm audit:create:prod
Copy-Item -Path "artifacts\create-project-audit" -Destination "artifacts\audit-dockerfile" -Recurse -Force
Remove-Item Env:\MYPAAS_AUDIT_REPO_URL

Write-Host "All fast audits complete!"
