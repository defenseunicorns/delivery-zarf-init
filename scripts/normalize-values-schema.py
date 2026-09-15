# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

"""Generate Zarf values schemas with temporary native-type fixes.

Remove when Zarf infers native types or the charts provide JSON schemas.
"""

import argparse
import json
import subprocess
from pathlib import Path
from typing import Any

# Types for templated values absent from chart defaults.
TYPE_OVERRIDES = {
    "gitea": {
        ("gitea", "config", "service", "DISABLE_REGISTRATION"): "boolean",
    },
    "zarf-registry": {
        ("proxy", "hostNetwork"): "boolean",
    },
}


def json_type(value: Any) -> str | None:
    if isinstance(value, bool):
        return "boolean"
    if isinstance(value, dict):
        return "object"
    if isinstance(value, list):
        return "array"
    if isinstance(value, int):
        return "integer"
    if isinstance(value, float):
        return "number"
    if isinstance(value, str):
        return "string"
    return None


def normalize(schema: dict[str, Any], defaults: Any) -> None:
    expected_type = json_type(defaults)
    if expected_type is None:
        schema.pop("type", None)
    elif "type" in schema:
        schema["type"] = expected_type

    if isinstance(defaults, dict):
        properties = schema.get("properties", {})
        for key, child_schema in properties.items():
            if key in defaults:
                normalize(child_schema, defaults[key])


def parse_yaml(document: str) -> Any:
    output = subprocess.check_output(
        ["uds", "zarf", "tools", "yq", "-o=json", "."],
        input=document,
        text=True,
    )
    return json.loads(output)


def load_yaml(path: Path) -> Any:
    return parse_yaml(path.read_text(encoding="utf-8"))


def yaml_scalar(value: Any) -> str:
    if isinstance(value, bool):
        return str(value).lower()
    if value is None:
        return ""
    return str(value)


def create_arguments(root: Path) -> list[str]:
    config = load_yaml(root / "flavors/upstream.yaml")
    settings = config["package"]["create"]["set"]
    arguments = []
    for key, value in settings.items():
        if key == "pkg_version":
            continue
        arguments.extend(("--set", f"{key.upper()}={yaml_scalar(value)}"))
    return arguments


def chart_field(document: Any, component_name: str, chart_name: str, field: str) -> Any:
    for component in document.get("components", []):
        if component.get("name") != component_name:
            continue
        for chart in component.get("charts", []):
            if chart.get("name") == chart_name:
                return chart[field]
    raise ValueError(f"chart {chart_name!r} not found in component {component_name!r}")


def command_output(command: list[str], root: Path) -> str:
    return subprocess.check_output(command, cwd=root, text=True)


def chart_defaults(root: Path) -> dict[str, Any]:
    upstream_gitea = load_yaml(root / ".zarf-src/packages/gitea/zarf.yaml")
    common = load_yaml(root / "components/common/zarf.yaml")
    gitea_url = chart_field(upstream_gitea, "git-server", "gitea", "url")
    gitea_version = chart_field(common, "git-server", "gitea", "version")
    gitea_values = command_output(
        [
            "uds",
            "zarf",
            "tools",
            "helm",
            "show",
            "values",
            str(gitea_url),
            "--version",
            str(gitea_version),
        ],
        root,
    )
    return {
        "zarf-agent": load_yaml(
            root / ".zarf-src/packages/zarf-agent/chart/values.yaml"
        ),
        "zarf-registry": load_yaml(
            root / ".zarf-src/packages/zarf-registry/chart/values.yaml"
        ),
        "gitea": parse_yaml(gitea_values),
    }


def set_type(schema: dict[str, Any], path: tuple[str, ...], expected_type: str) -> None:
    current = schema
    for segment in path:
        current = current["properties"][segment]
    current["type"] = expected_type


def normalized_schema(raw_schema: str, defaults: dict[str, Any]) -> str:
    schema = json.loads(raw_schema)
    for chart_name, chart_schema in schema.get("properties", {}).items():
        chart_values = defaults.get(chart_name)
        if chart_values is not None:
            normalize(chart_schema, chart_values)
        for path, expected_type in TYPE_OVERRIDES.get(chart_name, {}).items():
            set_type(chart_schema, path, expected_type)
    return f"{json.dumps(schema, indent=2)}\n"


def generate_schemas(root: Path, check: bool) -> int:
    defaults = chart_defaults(root)
    arguments = create_arguments(root)
    stale: list[Path] = []
    values_files = sorted((root / "packages").glob("*/zarf-values.yaml"))
    if not values_files:
        raise SystemExit("no zarf-values.yaml files found")

    for values_file in values_files:
        package_dir = values_file.parent
        schema_file = package_dir / "zarf-values.schema.json"
        relative_schema = schema_file.relative_to(root)
        print(f"{'Checking' if check else 'Generating'} {relative_schema}")
        raw_schema = command_output(
            [
                "uds",
                "zarf",
                "dev",
                "generate-schema",
                str(package_dir),
                "--flavor",
                "upstream",
                *arguments,
                "--delete-not-found",
            ],
            root,
        )
        generated = normalized_schema(raw_schema, defaults)
        if check:
            committed = (
                schema_file.read_text(encoding="utf-8")
                if schema_file.exists()
                else None
            )
            if committed != generated:
                stale.append(relative_schema)
        else:
            schema_file.write_text(generated, encoding="utf-8")

    for schema_file in stale:
        print(f"ERROR: {schema_file} is stale")
    if stale:
        print("Run: uds run generate-zarf-values-schemas")
        return 1

    return 0


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--check",
        action="store_true",
        help="fail if generated schemas differ from committed schemas",
    )
    args = parser.parse_args()
    root = Path(__file__).resolve().parent.parent
    return generate_schemas(root, args.check)


if __name__ == "__main__":
    raise SystemExit(main())
