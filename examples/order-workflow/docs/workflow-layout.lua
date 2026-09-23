-- Document-specific composition. Keep generated QMD and ddq's mechanism files
-- untouched while placing each workflow chapter on continuous landscape pages.
-- Otherwise every generated .landscape table starts a separate sheet, and its
-- preceding heading is stranded on a portrait page.
if not FORMAT:match("typst") then return {} end

local function unwrap_landscape(blocks)
  local result = pandoc.List()
  for _, block in ipairs(blocks) do
    if block.t == "Div" and block.classes:includes("landscape") then
      result:extend(unwrap_landscape(block.content))
    else
      result:insert(block)
    end
  end
  return result
end

return {{
  Pandoc = function(doc)
    local result = pandoc.List()
    local chapter = nil
    local function flush()
      if chapter then
        result:insert(pandoc.Div(unwrap_landscape(chapter), pandoc.Attr("", {"landscape"})))
        chapter = nil
      end
    end
    for _, block in ipairs(doc.blocks) do
      if block.t == "Header" and block.level == 1 then
        flush()
        if block.classes:includes("landscape-chapter") then
          chapter = pandoc.List()
        end
      end
      if chapter then chapter:insert(block) else result:insert(block) end
    end
    flush()
    doc.blocks = result
    return doc
  end
}}
