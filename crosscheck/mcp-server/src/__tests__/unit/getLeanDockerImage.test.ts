import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { getLeanDockerImage } from "../../docker.js";

describe("getLeanDockerImage", () => {
  let originalEnv: string | undefined;

  beforeEach(() => {
    originalEnv = process.env.LEAN_DOCKER_IMAGE;
  });

  afterEach(() => {
    if (originalEnv === undefined) {
      delete process.env.LEAN_DOCKER_IMAGE;
    } else {
      process.env.LEAN_DOCKER_IMAGE = originalEnv;
    }
  });

  it("returns the env var value when LEAN_DOCKER_IMAGE is set", () => {
    process.env.LEAN_DOCKER_IMAGE = "custom-lean:v2";
    expect(getLeanDockerImage()).toBe("custom-lean:v2");
  });

  it("returns the default image when LEAN_DOCKER_IMAGE is not set", () => {
    delete process.env.LEAN_DOCKER_IMAGE;
    expect(getLeanDockerImage()).toBe("crosscheck-lean:latest");
  });

  it("returns the default image when LEAN_DOCKER_IMAGE is an empty string", () => {
    process.env.LEAN_DOCKER_IMAGE = "";
    expect(getLeanDockerImage()).toBe("crosscheck-lean:latest");
  });
});
