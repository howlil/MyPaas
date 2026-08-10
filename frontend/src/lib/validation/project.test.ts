import { describe, expect, it } from "vitest";

import {
  validateProjectCreateInput,
  validateProjectUpdateInput,
} from "./project";

function validCreate(overrides: Record<string, unknown> = {}) {
  return {
    name: "demo-app",
    repoUrl: "https://github.com/example/demo",
    branch: "main",
    deployMode: "dockerfile",
    appPort: 3000,
    memoryLimitMb: 256,
    cpuLimit: 0.35,
    baseDirectory: null,
    staticFrontendPath: null,
    ...overrides,
  };
}

describe("validateProjectCreateInput", () => {
  it("accepts a backend-compatible project payload", () => {
    expect(() => validateProjectCreateInput(validCreate())).not.toThrow();
  });

  it("accepts uppercase names because the backend normalizes them", () => {
    expect(() =>
      validateProjectCreateInput(validCreate({ name: "Demo-App" })),
    ).not.toThrow();
  });

  it("rejects names that fail the backend shape", () => {
    expect(() =>
      validateProjectCreateInput(validCreate({ name: "-bad-" })),
    ).toThrow(/Project name/);
  });

  it("rejects out-of-range ports before making the request", () => {
    expect(() =>
      validateProjectCreateInput(validCreate({ appPort: 70000 })),
    ).toThrow(/App port/);
  });

  it("requires a main service for compose projects", () => {
    expect(() =>
      validateProjectCreateInput(
        validCreate({ deployMode: "compose", mainService: "" }),
      ),
    ).toThrow(/Main service/);
  });

  it("rejects repository path traversal", () => {
    expect(() =>
      validateProjectCreateInput(validCreate({ baseDirectory: "../api" })),
    ).toThrow(/parent-directory/);
  });

  it("rejects backslash-based repository paths", () => {
    expect(() =>
      validateProjectCreateInput(
        validCreate({ composeFilePath: "infra\\compose.yml" }),
      ),
    ).toThrow(/forward slashes/);
  });
});

describe("validateProjectUpdateInput", () => {
  it("allows a partial update", () => {
    expect(() =>
      validateProjectUpdateInput({ branch: "release" }),
    ).not.toThrow();
  });

  it("allows appPort=0 because PATCH uses it as preserve-current", () => {
    expect(() => validateProjectUpdateInput({ appPort: 0 })).not.toThrow();
  });

  it("validates paths when they are present", () => {
    expect(() =>
      validateProjectUpdateInput({ staticFrontendPath: "/absolute" }),
    ).toThrow(/repository root/);
  });
});
