#!/usr/bin/env python3

# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

"""Restore native Helm value types after Zarf schema generation."""

import argparse
import json
from pathlib import Path
from typing import Any

import yaml


# These values are introduced only by Zarf's templated override files, so they
# do not exist in the upstream chart defaults used by the recursive normalizer.
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
    if isinstance(value, (int, float)):
        return "number"
    if isinstance(value, str):
        return "string"
    return None


def normalize(schema: dict[str, Any], defaults: Any) -> None:
    expected_type = json_type(defaults)
    if expected_type is not None and "type" in schema:
        schema["type"] = expected_type

    if isinstance(defaults, dict):
        properties = schema.get("properties", {})
        for key, child_schema in properties.items():
            if key in defaults:
                normalize(child_schema, defaults[key])
    elif isinstance(defaults, list) and defaults:
        items = schema.get("items")
        if isinstance(items, dict):
            normalize(items, defaults[0])


def load_yaml(path: Path) -> Any:
    with path.open(encoding="utf-8") as stream:
        return yaml.safe_load(stream)


def set_type(schema: dict[str, Any], path: tuple[str, ...], expected_type: str) -> None:
    current = schema
    for segment in path:
        current = current["properties"][segment]
    current["type"] = expected_type


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("schema", type=Path)
    parser.add_argument("--agent-values", required=True, type=Path)
    parser.add_argument("--registry-values", required=True, type=Path)
    parser.add_argument("--gitea-values", required=True, type=Path)
    args = parser.parse_args()

    with args.schema.open(encoding="utf-8") as stream:
        schema = json.load(stream)

    chart_defaults = {
        "zarf-agent": load_yaml(args.agent_values),
        "zarf-registry": load_yaml(args.registry_values),
        "gitea": load_yaml(args.gitea_values),
    }

    for chart_name, chart_schema in schema.get("properties", {}).items():
        defaults = chart_defaults.get(chart_name)
        if defaults is not None:
            normalize(chart_schema, defaults)
        for path, expected_type in TYPE_OVERRIDES.get(chart_name, {}).items():
            set_type(chart_schema, path, expected_type)

    with args.schema.open("w", encoding="utf-8") as stream:
        json.dump(schema, stream, indent=2)
        stream.write("\n")


if __name__ == "__main__":
    main()
