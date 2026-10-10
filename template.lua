---@diagnostic disable: duplicate-doc-alias
---@alias Metadata { english_title: string?, romaji_title: string?, native_title: string?, synonyms: string[]?, community_score: number?, summary: string?, cover_image: string?, banner_image: string?, tag_list: string[]?, genre_list: string[]?, character_list: string[]?, author_list: string[]?, artist_list: string[]?, translator_list: string[]?, letterer_list: string[]?, date_start: string?, date_end: string?, provider_publisher: string?, publication_status: string?, publication_format: string?, country_of_origin: string?, chapter_count: number?, extra_notes: string?, source_url: string?, provider_id: string?, provider_id_code: string? }
---@alias Manga { id: string, title: string, url: string?, cover: string?, banner: string?, metadata: Metadata?, [any]: any }
---@alias Volume { number: number, [any]: any }
---@alias Chapter { title: string, url: string?, number: number?, date: string?, scanlation_group: string?, unavailable: string?, [any]: any }
---@alias Page { url: string, headers: table<string, string>?, cookies: table<string, string>?, extension: string? }

local sdk = require("sdk")

---@param query string
---@return Manga[]
function SearchMangas(query)
  return {}
end

---@param manga Manga
---@return Volume[]
function MangaVolumes(manga)
  return {}
end

---@param volume Volume
---@return Chapter[]
function VolumeChapters(volume)
  return {}
end

---@param chapter Chapter
---@return Page[]
function ChapterPages(chapter)
  return {}
end
