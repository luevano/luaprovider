# Lua Provider Gen

A utility tool to generate lua script based on templates.

## Install

```bash
go install .
```

## Usage

```
Usage of lua-provider-gen:
  -dir string
    	output directory (default ".")
  -luarc
    	generate .luarc.json file for language server configuration
  -provider
    	generate main.lua provider template (the file mangal loads)
  -sdk
    	generate sdk.lua file for language server
```
