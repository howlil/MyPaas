import { describe, expect, it } from "vitest";

import { projectNameValidationMessage, suggestProjectName } from "./project";

describe("guided project analysis naming", () => {
  it("derives a valid project name immediately from a pasted GitHub repository", () => {
    const name = suggestProjectName("https://github.com/example/my-service.git");

    expect(name).toBe("my-service");
    expect(projectNameValidationMessage(name)).toBe("");
  });

  it("keeps an empty source neutral until a repository is supplied", () => {
    expect(suggestProjectName("")).toBe("");
  });
});
