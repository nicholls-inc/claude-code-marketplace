import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { getDockerImage } from "../../docker.js";
describe("getDockerImage", () => {
    let originalEnv;
    beforeEach(() => {
        originalEnv = process.env.DAFNY_DOCKER_IMAGE;
    });
    afterEach(() => {
        if (originalEnv === undefined) {
            delete process.env.DAFNY_DOCKER_IMAGE;
        }
        else {
            process.env.DAFNY_DOCKER_IMAGE = originalEnv;
        }
    });
    it("returns the env var value when DAFNY_DOCKER_IMAGE is set", () => {
        process.env.DAFNY_DOCKER_IMAGE = "custom-dafny:v2";
        expect(getDockerImage()).toBe("custom-dafny:v2");
    });
    it("returns the default image when DAFNY_DOCKER_IMAGE is not set", () => {
        delete process.env.DAFNY_DOCKER_IMAGE;
        expect(getDockerImage()).toBe("crosscheck-dafny:latest");
    });
    it("returns the default image when DAFNY_DOCKER_IMAGE is an empty string", () => {
        process.env.DAFNY_DOCKER_IMAGE = "";
        expect(getDockerImage()).toBe("crosscheck-dafny:latest");
    });
});
