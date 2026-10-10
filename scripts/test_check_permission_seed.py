"""Guard against authorization regressions in permission seed changes."""

import copy
import json
from pathlib import Path
import unittest

from check_permission_seed import validate_seed


class PermissionSeedTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.seed = json.loads((Path(__file__).resolve().parents[1]
            / "backend/internal/data/seed/admin_permissions.json").read_text())

    def test_repository_seed(self):
        self.assertEqual(validate_seed(self.seed), (52, 47, 3))

    def test_invalid_authorization_changes(self):
        mutations = {
            "missing parent": lambda s: s["nodes"][1].update(parentCode="missing"),
            "cycle": lambda s: s["nodes"][1].update(parentCode=s["nodes"][1]["code"]),
            "duplicate code": lambda s: s["nodes"].append(copy.deepcopy(s["nodes"][0])),
            "unknown grant": lambda s: s["roles"][1]["permissions"].append("unknown.write"),
            "viewer writes": lambda s: s["roles"][2]["permissions"].append("media.upload"),
            "anonymous write": lambda s: s["anonymousApis"].append(
                {"method": "POST", "path": "/admin/api/v1/users"}),
            "duplicate route": lambda s: s["nodes"][5]["apis"].append(
                copy.deepcopy(s["nodes"][4]["apis"][0])),
            "public API in tree": lambda s: s["nodes"][4]["apis"][0].update(path="/api/v1/site"),
            "invalid scope": lambda s: s["selfService"][0]["allowedScopes"].append("anonymous"),
        }
        for name, mutate in mutations.items():
            with self.subTest(name=name):
                seed = copy.deepcopy(self.seed)
                mutate(seed)
                with self.assertRaises(ValueError):
                    validate_seed(seed)


if __name__ == "__main__":
    unittest.main()
