<div align="center">
  <img width="150px" alt="a logo depicting a moon" src="moon.png">
  <h1>Lua Provider</h1>
  <a href="https://pkg.go.dev/github.com/luevano/luaprovider"><img src="https://pkg.go.dev/badge/github.com/luevano/luaprovider.svg" alt="Go Reference"></a>
</div>

> **Warning**
>
> The API is not stable and may change at any time.

This is a generic provider for [libmangal](https://github.com/luevano/libmangal) that uses [Lua](https://www.lua.org/) scripts to create _subproviders_.

It uses [native go implementation of the Lua interpreter](https://github.com/yuin/gopher-lua) and provides a set of libraries that can be used in the scripts.

**Note**: This is a fork of [mangalorg/luaprovider](https://github.com/mangalorg/luaprovider).

> **Warning**
>
> Provider scripts are code: they run with your network access (http, headless browsers) and the
> SDK libraries. Scripts get no `io`/`os` and `require` only loads files from the provider's own
> package paths, but only install providers you trust. Requests aren't limited to the internet:
> a script can also reach `localhost` and your LAN (a router, a FlareSolverr, other local
> services) unless the app guards the `HTTPClient` it passes (mangal does, see its
> `providers.allow_private_addresses`). Headless browsers aren't covered by that.

## Features

- Built-in Lua VM without CGO with
  [gopher-lua](https://github.com/yuin/gopher-lua)
- Batteries-included library
- Ships with a [CLI helper tool](./cmd/lua-provider-gen) for generating script templates
- Luadoc generation which enables autocompletion for your IDE

> **Note**
>
> It is recommended to use
> [lua-language-server](https://github.com/LuaLS/lua-language-server) to get
> nice completions for your IDE
>
> [VSCode extension](https://marketplace.visualstudio.com/items?itemName=sumneko.lua)

## Usage

```go
script, err := os.ReadFile("main.lua")
if err != nil {
	panic(err)
}
info := libmangal.ProviderInfo{ID: "weebcentral", Name: "Weeb Central", Version: "0.3.1"}

loader, err := luaprovider.NewLoader(script, info, luaprovider.DefaultOptions())
if err != nil {
	panic(err)
}
client, err := libmangal.NewClient(loader, libmangal.DefaultClientOptions())
if err != nil {
	panic(err)
}
defer client.Close()

mangas, err := client.SearchMangas("dandadan")
```

[mangal](https://github.com/luevano/mangal) loads scripts from its providers directory, with the info
from a `mangal.toml` next to each script (see [saturno](https://github.com/luevano/saturno)).

## Scripts

### Overview

See [examples of scripts](https://github.com/luevano/saturno).

See the [SDK documentation](./docs/sdk.lua).

Scripts must look like this:

```lua
function SearchMangas(query)
  return {}
end

function MangaVolumes(manga)
 return {}
end

function VolumeChapters(volume)
 return {}
end

function ChapterPages(chapter)
  return {}
end
```

Notice the four required global functions

- `SearchMangas` - searches for mangas based on the given query.
- `MangaVolumes` - gets manga volumes. Each manga must have at least 1 volume.
- `VolumeChapters` - gets chapters of the given volume.
- `ChapterPages` - gets pages of the given chapter.

See the [template](./template.lua) for the fields each of them can return.

Optionally, `MangaByID(id)` gets the manga with the given id (the `id` its search results have),
or returns `nil` if there's none (a not found `response:check()` error counts as none too). Apps
use it to get a manga again without searching; without it they search and match the id.

Errors raised with `error()` are plain messages. For HTTP errors, `response:check()` raises one the app can tell apart (not found, rate limited), and `send` already does for 429.

If the site documents a rate limit, set it when the script loads with
`sdk.http.rate_limit(permits, seconds[, match[, burst]])`, e.g. `sdk.http.rate_limit(1, 2, "site.com")`
for one request every 2 seconds to `site.com`. Requests (page images too) wait for every limit they match.

For a site behind a Cloudflare challenge, `request:flaresolverr()` sends that request through
[FlareSolverr](https://github.com/FlareSolverr/FlareSolverr) when the app has one configured
(`sdk.http.has_flaresolverr()`, else it raises), e.g.
`sdk.http.request("GET", url):flaresolverr():send()`. Only GET and POST; FlareSolverr doesn't pass
the request's headers or cookies, and json comes back inside the page's `<pre>`. It's slow, so use
it only where needed. Apps set `Options.FlareSolverr` to a function making the transport (e.g.
mangoprovider's `flaresolverr.NewTransportClient`); one session per loaded provider, closed with it.

---

The scripts can load sdk with

```lua
local sdk = require("sdk")
```

Which provides these packages:

[Documentation](./docs/sdk.lua).

<details>
<summary>Packages</summary>

- [http](./lib/http)
- [headless](./lib/headless)
- [crypto](./lib/crypto)
  - [md5](./lib/crypto/md5)
  - [sha1](./lib/crypto/sha1)
  - [sha256](./lib/crypto/sha256)
  - [sha512](./lib/crypto/sha512)
  - [aes](./lib/crypto/aes)
- [encoding](./lib/encoding)
  - [base64](./lib/encoding/base64)
  - [json](./lib/encoding/json)
- [html](./lib/html)
- [js](./lib/js)
- [regexp](./lib/regexp)
- [time](./lib/time)
- [strings](./lib/strings)
- [levenshtein](./lib/levenshtein)
- [util](./lib/util)
- [urls](./lib/urls)
- [log](./lib/log)
- [options](./lib/options)
- [settings](./lib/settings)

</details>

### Script Development

Install the [helper tool](./cmd/lua-provider-gen)

```bash
just install-cmd
```

Then use it to generate a new workflow

```bash
lua-provider-gen -sdk -provider -luarc
```

This command will create the following files:

- `sdk.lua` - gives IDE autocompletion
- `main.lua` - a provider script template (mangal loads `main.lua` next to a `mangal.toml`)
- `.luarc.json` - a language server configuration

## Credits

Moon logo: [Moon phase icons created by vectorsmarket15 - Flaticon](https://www.flaticon.com/free-icons/moon-phase "moon phase icons")
