"""Distribution contract checks using anonymous files and temporary directories."""
import hashlib
import importlib.util
from pathlib import Path
import tempfile
import unittest
import zipfile

spec = importlib.util.spec_from_file_location("package_release", Path(__file__).with_name("package-release.py"))
release = importlib.util.module_from_spec(spec)
spec.loader.exec_module(release)


class PackageReleaseTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.binaries = self.root / "dist"
        self.binaries.mkdir()
        self.output = self.root / "release"
        self.commit = "a" * 40
        self.sources = {}
        for os_name, arch in release.TARGETS:
            extension = ".exe" if os_name == "windows" else ""
            path = self.binaries / f"fegan-master-team-{os_name}-{arch}{extension}"
            content = f"anonymous executable {os_name}/{arch}".encode()
            path.write_bytes(content)
            self.sources[(os_name, arch)] = content
        (self.binaries / "team.sqlite").write_bytes(b"must never be packaged")
        (self.binaries / "unexpected.txt").write_text("unrelated")

    def test_archives_contain_only_target_binary_and_guide_with_valid_checksums(self):
        release.package("v1.2.3", self.commit, self.binaries, self.output)
        self.assertEqual(len(list(self.output.iterdir())), 7)
        hashes = {}
        for line in (self.output / "SHA256SUMS").read_text().splitlines():
            digest, name = line.split("  ")
            self.assertEqual(digest, hashlib.sha256((self.output / name).read_bytes()).hexdigest())
            hashes[name] = digest
        self.assertEqual(len(hashes), 6)
        for os_name, arch in release.TARGETS:
            name = f"fegan-master-team-v1.2.3-{os_name}-{arch}"
            extension = ".exe" if os_name == "windows" else ""
            with zipfile.ZipFile(self.output / f"{name}.zip") as archive:
                binary, guide = f"{name}/fegan-master-team{extension}", f"{name}/LEEME.txt"
                self.assertEqual(set(archive.namelist()), {binary, guide})
                self.assertEqual(archive.read(binary), self.sources[(os_name, arch)])
                self.assertEqual(archive.getinfo(binary).external_attr >> 16, 0o100755)
                text = archive.read(guide).decode("utf-8")
                for expected in ("v1.2.3", self.commit, f"{os_name}/{arch}", "-no-browser", "SHA256SUMS"):
                    self.assertIn(expected, text)
                self.assertIsNone(archive.testzip())

    def test_rejects_bad_version_or_commit_before_creating_output(self):
        for version, commit in [("../v1.2.3", self.commit), ("v01.2.3", self.commit),
                                ("v1.2.3-rc.1", self.commit), ("v1.2.3", "branch")]:
            with self.subTest(version=version, commit=commit):
                with self.assertRaises(ValueError):
                    release.package(version, commit, self.binaries, self.output)
                self.assertFalse(self.output.exists())

    def test_missing_binary_creates_no_partial_release(self):
        next(self.binaries.glob("*.exe")).unlink()
        with self.assertRaises(ValueError):
            release.package("v1.2.3", self.commit, self.binaries, self.output)
        self.assertFalse(self.output.exists())

    def test_existing_output_is_preserved(self):
        self.output.mkdir()
        existing = self.output / "already.zip"
        existing.write_bytes(b"existing release")
        with self.assertRaises(ValueError):
            release.package("v1.2.3", self.commit, self.binaries, self.output)
        self.assertEqual(existing.read_bytes(), b"existing release")
        self.assertEqual(list(self.output.iterdir()), [existing])


if __name__ == "__main__":
    unittest.main()
