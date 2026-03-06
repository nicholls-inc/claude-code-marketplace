import { describe, it, expect } from "vitest";
import { stripBoilerplate } from "../../tools/compile.js";

describe("stripBoilerplate", () => {
  describe("Python target", () => {
    it("strips 'from _dafny import ...' lines", () => {
      const input = "from _dafny import Module\nprint('hello')\n";
      const result = stripBoilerplate(input, "py");
      expect(result).not.toContain("from _dafny import");
      expect(result).toContain("print('hello')");
    });

    it("strips 'import _dafny' lines", () => {
      const input = "import _dafny\nprint('hello')\n";
      const result = stripBoilerplate(input, "py");
      expect(result).not.toContain("import _dafny");
      expect(result).toContain("print('hello')");
    });

    it("strips 'import _System' lines", () => {
      const input = "import _System\nprint('hello')\n";
      const result = stripBoilerplate(input, "py");
      expect(result).not.toContain("import _System");
      expect(result).toContain("print('hello')");
    });

    it("strips 'from _System import ...' lines", () => {
      const input = "from _System import Foo\nprint('hello')\n";
      const result = stripBoilerplate(input, "py");
      expect(result).not.toContain("from _System import");
      expect(result).toContain("print('hello')");
    });
  });

  describe("Go target", () => {
    it("strips _dafny import lines", () => {
      const input = '  _dafny "some/path/dafny"\nfunc main() {}\n';
      const result = stripBoilerplate(input, "go");
      expect(result).not.toContain("_dafny");
      expect(result).toContain("func main() {}");
    });

    it("strips _System import lines", () => {
      const input = '  _System "some/path/System_"\nfunc main() {}\n';
      const result = stripBoilerplate(input, "go");
      expect(result).not.toContain("_System");
      expect(result).toContain("func main() {}");
    });
  });

  it("collapses 3+ consecutive newlines to 2 after stripping", () => {
    const input = "line1\n\n\n\nline2\n";
    const result = stripBoilerplate(input, "py");
    expect(result).not.toMatch(/\n{3,}/);
    expect(result).toContain("line1\n\nline2");
  });

  it("always ends with exactly one trailing newline", () => {
    const result1 = stripBoilerplate("hello", "py");
    expect(result1).toBe("hello\n");

    const result2 = stripBoilerplate("hello\n\n\n", "py");
    expect(result2).toBe("hello\n");
  });

  it("passes clean content through unchanged (except trailing newline normalization)", () => {
    const input = "def main():\n    pass";
    const result = stripBoilerplate(input, "py");
    expect(result).toBe("def main():\n    pass\n");
  });
});
