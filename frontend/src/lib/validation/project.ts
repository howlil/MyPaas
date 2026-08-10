const PROJECT_NAME_PATTERN = /^[a-z0-9][a-z0-9-]{1,28}[a-z0-9]$/;

function asRecord(input: unknown): Record<string, unknown> {
  if (!input || typeof input !== "object" || Array.isArray(input)) {
    throw new Error("Project payload must be an object");
  }
  return input as Record<string, unknown>;
}

function hasOwn(record: Record<string, unknown>, key: string) {
  return Object.prototype.hasOwnProperty.call(record, key);
}

function validateName(record: Record<string, unknown>, required: boolean) {
  if (!hasOwn(record, "name")) {
    if (required) throw new Error("Project name is required");
    return;
  }
  if (typeof record.name !== "string") {
    throw new Error("Project name must be a string");
  }
  const normalized = record.name.trim().toLowerCase();
  if (!PROJECT_NAME_PATTERN.test(normalized)) {
    throw new Error(
      "Project name must be 3-30 characters, use only letters, numbers, or dashes, and start/end with a letter or number",
    );
  }
}

function validateRequiredString(
  record: Record<string, unknown>,
  key: string,
  label: string,
) {
  const value = record[key];
  if (typeof value !== "string" || !value.trim()) {
    throw new Error(`${label} is required`);
  }
}

function validateRepoRelativePathValue(value: unknown, label: string) {
  if (value === null || value === undefined || value === "") return;
  if (typeof value !== "string") {
    throw new Error(`${label} must be a repository-relative path`);
  }
  const path = value.trim();
  if (!path) return;
  if (path.includes("\0")) {
    throw new Error(`${label} cannot contain NUL characters`);
  }
  if (path.startsWith("/") || path.startsWith("\\")) {
    throw new Error(`${label} must be relative to the repository root`);
  }
  if (path.includes("\\")) {
    throw new Error(`${label} must use forward slashes`);
  }
  if (path.split("/").some((segment) => segment === "..")) {
    throw new Error(`${label} cannot contain parent-directory segments`);
  }
}

function validateRepoRelativePath(
  record: Record<string, unknown>,
  key: string,
  label: string,
) {
  if (!hasOwn(record, key)) return;
  validateRepoRelativePathValue(record[key], label);
}

function validateComposeOverridePaths(record: Record<string, unknown>) {
  if (!hasOwn(record, "composeOverridePaths")) return;
  const value = record.composeOverridePaths;
  if (value === null || value === undefined) return;
  if (!Array.isArray(value)) {
    throw new Error("Compose override paths must be an array");
  }
  for (const path of value) {
    validateRepoRelativePathValue(path, "Compose override path");
  }
}

function validateNonNegativeNumber(
  record: Record<string, unknown>,
  key: string,
  label: string,
) {
  if (!hasOwn(record, key) || record[key] === null || record[key] === undefined)
    return;
  const value = record[key];
  if (typeof value !== "number" || !Number.isFinite(value)) {
    throw new Error(`${label} must be a finite number`);
  }
  if (value < 0) {
    throw new Error(`${label} must be zero or greater`);
  }
}

function validatePort(
  record: Record<string, unknown>,
  required: boolean,
  allowZero = false,
) {
  if (!hasOwn(record, "appPort")) {
    if (required) throw new Error("App port is required");
    return;
  }
  const value = record.appPort;
  const minimum = allowZero ? 0 : 1;
  if (
    typeof value !== "number" ||
    !Number.isInteger(value) ||
    value < minimum ||
    value > 65535
  ) {
    throw new Error(`App port must be an integer between ${minimum} and 65535`);
  }
}

function validateCommon(record: Record<string, unknown>) {
  validateRepoRelativePath(record, "baseDirectory", "Base directory");
  validateRepoRelativePath(
    record,
    "staticFrontendPath",
    "Static frontend path",
  );
  validateRepoRelativePath(record, "composeFilePath", "Compose file path");
  validateRepoRelativePath(
    record,
    "composeWorkdir",
    "Compose working directory",
  );
  validateComposeOverridePaths(record);
  validateNonNegativeNumber(record, "memoryLimitMb", "Memory limit");
  validateNonNegativeNumber(record, "memoryMb", "Memory limit");
  validateNonNegativeNumber(record, "cpuLimit", "CPU limit");
}

export function validateProjectCreateInput(input: unknown): void {
  const record = asRecord(input);
  validateName(record, true);
  validateRequiredString(record, "repoUrl", "Repository URL");
  validateCommon(record);

  if (hasOwn(record, "branch") && record.branch !== "") {
    validateRequiredString(record, "branch", "Branch");
  }

  const deployMode =
    typeof record.deployMode === "string" ? record.deployMode : "";
  if (deployMode === "compose") {
    validateRequiredString(record, "mainService", "Main service");
  }
  if (deployMode !== "static") {
    validatePort(record, true);
  } else if (hasOwn(record, "appPort")) {
    validatePort(record, false, true);
  }
}

export function validateProjectUpdateInput(input: unknown): void {
  const record = asRecord(input);
  validateName(record, false);
  validateCommon(record);
  if (hasOwn(record, "appPort")) {
    validatePort(record, false, true);
  }
}
