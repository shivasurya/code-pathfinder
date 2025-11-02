#!/usr/bin/env python3
"""
Comprehensive tests for generate_stdlib_registry.py

Tests cover:
- Function introspection (signatures, return types, docstrings)
- Class introspection (methods, docstrings)
- Constant introspection (types, values)
- Attribute introspection (dict-like, list-like behaviors)
- Manifest generation (checksums, statistics)
- End-to-end generation for all modules
"""

import json
import sys
import tempfile
import unittest
from pathlib import Path

# Import the generator module
sys.path.insert(0, str(Path(__file__).parent))
import generate_stdlib_registry as gen


class TestTypeNameConversion(unittest.TestCase):
    """Test get_type_name function."""

    def test_none_returns_unknown(self):
        """Test that None annotation returns 'unknown'."""
        self.assertEqual(gen.get_type_name(None), "unknown")

    def test_string_annotation(self):
        """Test string annotation passthrough."""
        self.assertEqual(gen.get_type_name("MyType"), "MyType")

    def test_builtin_type(self):
        """Test builtin type conversion."""
        self.assertEqual(gen.get_type_name(str), "builtins.str")
        self.assertEqual(gen.get_type_name(int), "builtins.int")
        self.assertEqual(gen.get_type_name(list), "builtins.list")

    def test_generic_type(self):
        """Test generic type handling (List[str], Dict[str, int])."""
        from typing import List, Dict

        # List[str]
        list_str_type = gen.get_type_name(List[str])
        self.assertIn("list", list_str_type.lower())

        # Dict[str, int]
        dict_type = gen.get_type_name(Dict[str, int])
        self.assertIn("dict", dict_type.lower())


class TestDocstringCleaning(unittest.TestCase):
    """Test clean_docstring function."""

    def test_empty_docstring(self):
        """Test empty docstring returns empty string."""
        self.assertEqual(gen.clean_docstring(""), "")
        self.assertEqual(gen.clean_docstring(None), "")

    def test_strip_whitespace(self):
        """Test leading/trailing whitespace is removed."""
        docstring = "  \n  Test docstring  \n  "
        self.assertEqual(gen.clean_docstring(docstring), "Test docstring")

    def test_remove_indentation(self):
        """Test common indentation is removed."""
        docstring = """First line.
        Second line.
        Third line."""
        cleaned = gen.clean_docstring(docstring)
        self.assertIn("First line", cleaned)
        self.assertIn("Second line", cleaned)
        self.assertNotIn("        Second", cleaned)

    def test_truncate_long_docstring(self):
        """Test long docstrings are truncated."""
        long_doc = "a" * 600
        cleaned = gen.clean_docstring(long_doc)
        self.assertEqual(len(cleaned), 500)
        self.assertTrue(cleaned.endswith("..."))


class TestFunctionIntrospection(unittest.TestCase):
    """Test introspect_function function."""

    def test_function_with_annotations(self):
        """Test function with type annotations."""
        def example_func(x: int, y: str) -> bool:
            """Example function."""
            return True

        result = gen.introspect_function(example_func)

        self.assertEqual(result["return_type"], "builtins.bool")
        self.assertEqual(result["confidence"], 1.0)
        self.assertEqual(result["source"], "annotation")
        self.assertEqual(len(result["params"]), 2)
        self.assertEqual(result["params"][0]["name"], "x")
        self.assertEqual(result["params"][0]["type"], "builtins.int")

    def test_function_without_annotations(self):
        """Test function without type annotations."""
        def no_annotations(x, y):
            """Return str."""
            return "test"

        result = gen.introspect_function(no_annotations)

        # Should infer from docstring or use unknown
        self.assertIn(result["return_type"], ["builtins.str", "unknown"])
        self.assertLessEqual(result["confidence"], 0.7)

    def test_function_with_defaults(self):
        """Test function with default parameters."""
        def with_defaults(x: int, y: str = "default") -> None:
            """Function with defaults."""
            pass

        result = gen.introspect_function(with_defaults)

        self.assertEqual(len(result["params"]), 2)
        self.assertTrue(result["params"][0]["required"])
        self.assertFalse(result["params"][1]["required"])

    def test_builtin_function(self):
        """Test builtin function introspection."""
        result = gen.introspect_function(len)

        # Builtins may not have signature
        self.assertIsNotNone(result)
        self.assertIn("return_type", result)

    def test_function_docstring(self):
        """Test function docstring extraction."""
        def documented():
            """This is a test docstring."""
            pass

        result = gen.introspect_function(documented)

        self.assertIn("docstring", result)
        self.assertEqual(result["docstring"], "This is a test docstring.")


class TestClassIntrospection(unittest.TestCase):
    """Test introspect_class function."""

    def test_simple_class(self):
        """Test simple class with methods."""
        class ExampleClass:
            """Example class."""
            def method1(self) -> int:
                """Method 1."""
                return 1

            def method2(self, x: str) -> str:
                """Method 2."""
                return x

        result = gen.introspect_class(ExampleClass)

        self.assertEqual(result["type"], "class")
        self.assertIn("methods", result)
        self.assertIn("method1", result["methods"])
        self.assertIn("method2", result["methods"])
        self.assertIn("docstring", result)

    def test_class_with_special_methods(self):
        """Test class with __init__, __call__, etc."""
        class SpecialClass:
            """Class with special methods."""
            def __init__(self, value: int):
                """Initialize."""
                self.value = value

            def __call__(self) -> int:
                """Call method."""
                return self.value

        result = gen.introspect_class(SpecialClass)

        self.assertIn("__init__", result["methods"])
        self.assertIn("__call__", result["methods"])

    def test_class_without_docstring(self):
        """Test class without docstring."""
        class NoDocClass:
            def method(self):
                pass

        result = gen.introspect_class(NoDocClass)

        self.assertEqual(result["type"], "class")
        # Docstring may not be present
        if "docstring" in result:
            self.assertEqual(result["docstring"], "")


class TestConstantIntrospection(unittest.TestCase):
    """Test introspect_constant function."""

    def test_string_constant(self):
        """Test string constant."""
        result = gen.introspect_constant("test string")

        self.assertEqual(result["type"], "builtins.str")
        self.assertEqual(result["value"], "'test string'")
        self.assertEqual(result["confidence"], 1.0)

    def test_int_constant(self):
        """Test integer constant."""
        result = gen.introspect_constant(42)

        self.assertEqual(result["type"], "builtins.int")
        self.assertEqual(result["value"], "42")
        self.assertEqual(result["confidence"], 1.0)

    def test_tuple_constant(self):
        """Test tuple constant."""
        result = gen.introspect_constant((1, 2, 3))

        self.assertEqual(result["type"], "builtins.tuple")
        self.assertIn("1", result["value"])
        self.assertEqual(result["confidence"], 1.0)

    def test_long_value_truncation(self):
        """Test long constant values are truncated."""
        long_string = "a" * 200
        result = gen.introspect_constant(long_string)

        self.assertTrue(len(result["value"]) <= 100)
        self.assertTrue(result["value"].endswith("..."))


class TestAttributeIntrospection(unittest.TestCase):
    """Test introspect_attribute function."""

    def test_dict_like_attribute(self):
        """Test dict-like attribute detection."""
        test_dict = {"key": "value"}
        result = gen.introspect_attribute(test_dict)

        self.assertEqual(result["behaves_like"], "builtins.dict")
        self.assertEqual(result["confidence"], 0.9)

    def test_list_like_attribute(self):
        """Test list-like attribute detection."""
        test_list = [1, 2, 3]
        result = gen.introspect_attribute(test_list)

        self.assertEqual(result["behaves_like"], "builtins.list")
        self.assertEqual(result["confidence"], 0.9)

    def test_custom_object(self):
        """Test custom object attribute."""
        class CustomObject:
            """Custom object."""
            pass

        obj = CustomObject()
        result = gen.introspect_attribute(obj)

        self.assertIn("type", result)
        self.assertIn("confidence", result)


class TestModuleIntrospection(unittest.TestCase):
    """Test introspect_module function."""

    def test_os_module(self):
        """Test introspection of os module."""
        result = gen.introspect_module("os")

        self.assertIsNotNone(result)
        self.assertEqual(result["module"], "os")
        self.assertIn("python_version", result)
        self.assertIn("functions", result)
        self.assertIn("classes", result)
        self.assertIn("constants", result)
        self.assertIn("attributes", result)

        # os should have functions like getcwd
        self.assertIn("getcwd", result["functions"])

        # os should have constants like sep, pathsep
        self.assertTrue(len(result["constants"]) > 0)

        # os should have attributes like environ
        self.assertIn("environ", result["attributes"])

    def test_sys_module(self):
        """Test introspection of sys module."""
        result = gen.introspect_module("sys")

        self.assertIsNotNone(result)
        self.assertEqual(result["module"], "sys")

        # sys should have functions like exit
        self.assertIn("exit", result["functions"])

        # sys should have attributes like modules, path
        self.assertIn("modules", result["attributes"])
        self.assertIn("path", result["attributes"])

    def test_json_module(self):
        """Test introspection of json module."""
        result = gen.introspect_module("json")

        self.assertIsNotNone(result)
        self.assertEqual(result["module"], "json")

        # json should have functions like dumps, loads
        self.assertIn("dumps", result["functions"])
        self.assertIn("loads", result["functions"])

        # json should have classes like JSONEncoder, JSONDecoder
        self.assertIn("JSONEncoder", result["classes"])
        self.assertIn("JSONDecoder", result["classes"])

    def test_invalid_module(self):
        """Test introspection of non-existent module."""
        result = gen.introspect_module("nonexistent_module_12345")

        self.assertIsNone(result)

    def test_private_members_excluded(self):
        """Test that private members (starting with _) are excluded."""
        result = gen.introspect_module("os")

        self.assertIsNotNone(result)

        # Check no private names in any category
        for func_name in result["functions"].keys():
            self.assertFalse(func_name.startswith("_"))

        for class_name in result["classes"].keys():
            self.assertFalse(class_name.startswith("_"))


class TestManifestGeneration(unittest.TestCase):
    """Test generate_manifest function."""

    def test_manifest_structure(self):
        """Test manifest has correct structure."""
        with tempfile.TemporaryDirectory() as tmpdir:
            output_dir = Path(tmpdir)

            # Generate a test module
            data = gen.introspect_module("json")
            output_file = output_dir / "json_stdlib.json"
            with open(output_file, "w") as f:
                json.dump(data, f)

            # Generate manifest
            manifest = gen.generate_manifest(output_dir, (3, 14, 0), ["json"])

            self.assertEqual(manifest["schema_version"], "1.0.0")
            self.assertEqual(manifest["registry_version"], "v1")
            self.assertEqual(manifest["python_version"]["major"], 3)
            self.assertEqual(manifest["python_version"]["minor"], 14)
            self.assertIn("generated_at", manifest)
            self.assertIn("base_url", manifest)
            self.assertIn("modules", manifest)
            self.assertIn("statistics", manifest)

    def test_manifest_checksums(self):
        """Test manifest includes checksums for modules."""
        with tempfile.TemporaryDirectory() as tmpdir:
            output_dir = Path(tmpdir)

            # Generate a test module
            data = gen.introspect_module("json")
            output_file = output_dir / "json_stdlib.json"
            with open(output_file, "w") as f:
                json.dump(data, f)

            # Generate manifest
            manifest = gen.generate_manifest(output_dir, (3, 14, 0), ["json"])

            self.assertEqual(len(manifest["modules"]), 1)
            module_entry = manifest["modules"][0]

            self.assertEqual(module_entry["name"], "json")
            self.assertEqual(module_entry["file"], "json_stdlib.json")
            self.assertIn("checksum", module_entry)
            self.assertTrue(module_entry["checksum"].startswith("sha256:"))
            self.assertIn("size_bytes", module_entry)
            self.assertGreater(module_entry["size_bytes"], 0)

    def test_manifest_statistics(self):
        """Test manifest includes accurate statistics."""
        with tempfile.TemporaryDirectory() as tmpdir:
            output_dir = Path(tmpdir)

            # Generate multiple test modules
            for module_name in ["json", "os", "sys"]:
                data = gen.introspect_module(module_name)
                if data:
                    output_file = output_dir / f"{module_name}_stdlib.json"
                    with open(output_file, "w") as f:
                        json.dump(data, f)

            # Generate manifest
            manifest = gen.generate_manifest(output_dir, (3, 14, 0), ["json", "os", "sys"])

            stats = manifest["statistics"]
            self.assertEqual(stats["total_modules"], 3)
            self.assertGreater(stats["total_functions"], 0)
            self.assertGreater(stats["total_classes"], 0)


class TestGetAllStdlibModules(unittest.TestCase):
    """Test get_all_stdlib_modules function."""

    def test_returns_list(self):
        """Test that function returns a list."""
        modules = gen.get_all_stdlib_modules()
        self.assertIsInstance(modules, list)

    def test_no_private_modules(self):
        """Test that no private modules are included."""
        modules = gen.get_all_stdlib_modules()
        for module_name in modules:
            self.assertFalse(module_name.startswith("_"))

    def test_sorted_alphabetically(self):
        """Test that modules are sorted alphabetically."""
        modules = gen.get_all_stdlib_modules()
        self.assertEqual(modules, sorted(modules))

    def test_contains_common_modules(self):
        """Test that common modules are included."""
        modules = gen.get_all_stdlib_modules()
        for common in ["os", "sys", "json", "pathlib", "datetime"]:
            self.assertIn(common, modules)


class TestEndToEnd(unittest.TestCase):
    """Test end-to-end generation."""

    def test_generate_small_set(self):
        """Test generating a small set of modules."""
        with tempfile.TemporaryDirectory() as tmpdir:
            output_dir = Path(tmpdir)

            test_modules = ["json", "os", "sys"]
            successful = []

            for module_name in test_modules:
                data = gen.introspect_module(module_name)
                if data:
                    output_file = output_dir / f"{module_name}_stdlib.json"
                    with open(output_file, "w") as f:
                        json.dump(data, f)
                    successful.append(module_name)

            # Generate manifest
            manifest = gen.generate_manifest(output_dir, sys.version_info[:3], successful)

            manifest_file = output_dir / "manifest.json"
            with open(manifest_file, "w") as f:
                json.dump(manifest, f)

            # Verify all files exist
            self.assertTrue(manifest_file.exists())
            for module_name in successful:
                self.assertTrue((output_dir / f"{module_name}_stdlib.json").exists())

            # Verify manifest
            self.assertEqual(manifest["statistics"]["total_modules"], len(successful))

    def test_generated_json_valid(self):
        """Test that generated JSON is valid and loadable."""
        with tempfile.TemporaryDirectory() as tmpdir:
            output_dir = Path(tmpdir)

            # Generate module
            data = gen.introspect_module("json")
            output_file = output_dir / "json_stdlib.json"
            with open(output_file, "w") as f:
                json.dump(data, f)

            # Try to load it back
            with open(output_file) as f:
                loaded = json.load(f)

            self.assertEqual(loaded["module"], "json")
            self.assertIn("functions", loaded)
            self.assertIn("classes", loaded)


def run_tests():
    """Run all tests."""
    loader = unittest.TestLoader()
    suite = loader.loadTestsFromModule(sys.modules[__name__])
    runner = unittest.TextTestRunner(verbosity=2)
    result = runner.run(suite)
    return 0 if result.wasSuccessful() else 1


if __name__ == "__main__":
    sys.exit(run_tests())
