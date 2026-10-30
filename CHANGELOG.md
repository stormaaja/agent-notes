# Changelog

All notable changes to this project are documented in this file. It is generated from
[Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) by `scripts/release.sh`;
do not edit it by hand.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this
project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2026-10-30

### Added

- Add pi skill for agent-notes ([a6507db](https://github.com/stormaaja/agent-notes/commit/a6507db824214d23a3f416b1a50eb94944d3ffab))

## [0.1.0] - 2026-10-28

### Added

- **note:** Scaffold module and add slug and tag helpers ([33861ee](https://github.com/stormaaja/agent-notes/commit/33861ee7635762416b359f354b19abc3fae42860))
- **note:** Parse and serialize note frontmatter ([c03055a](https://github.com/stormaaja/agent-notes/commit/c03055aec503cb91c3d016607b30d8bc9f4e94d0))
- **store:** Resolve the notes directory ([8df9ff8](https://github.com/stormaaja/agent-notes/commit/8df9ff8cdd39fa2c68f0d39a24b8af59c9954c2a))
- **store:** Create and get notes with atomic writes ([d0f8e62](https://github.com/stormaaja/agent-notes/commit/d0f8e627c2d4032808bdbb88befa8ac54cd419a1))
- **store:** List notes ([21dc838](https://github.com/stormaaja/agent-notes/commit/21dc838029a9d568de00710d07bc72b1e7a1257f))
- **store:** Update and delete notes ([df5083a](https://github.com/stormaaja/agent-notes/commit/df5083a079bb9444f8a0b74d1f0f7dd64713a45d))
- **search:** Add tag and keyword search with ranking ([18ba9a7](https://github.com/stormaaja/agent-notes/commit/18ba9a707fd4f8d4c04eebffed8a2e8af429cdae))
- **cli:** Add command dispatch and init command ([b2758d4](https://github.com/stormaaja/agent-notes/commit/b2758d41f6e3e40ab4b2e11cebe2e8ca7847bab6))
- **cli:** Add add and show commands ([6d9319e](https://github.com/stormaaja/agent-notes/commit/6d9319e99e64c5d3cbddd47b08fdcf3589cc596c))
- **cli:** Add list and search commands ([37ccb14](https://github.com/stormaaja/agent-notes/commit/37ccb14f3327194a7245c6a824ef3e7f3f2e80c4))
- **cli:** Add update and delete commands ([4d5e658](https://github.com/stormaaja/agent-notes/commit/4d5e65886b33a7b529be62b40217c484036484b2))
- **cli:** Add instructions command ([a8c3411](https://github.com/stormaaja/agent-notes/commit/a8c3411458ddf50588f082cf0bd558a057e759dc))
- **memory:** Parse Claude Code memory files ([d64bcaa](https://github.com/stormaaja/agent-notes/commit/d64bcaafdea8c8f158ceace113b8695265e09853))
- **cli:** Add import-memory command ([aad0ade](https://github.com/stormaaja/agent-notes/commit/aad0ade8bad5b3340fc81f07ec93e2e69177989e))
- Add Claude Code skill for agent-notes ([0d8b27d](https://github.com/stormaaja/agent-notes/commit/0d8b27d01e5244a2617a58b17f73bd1e04e6221d))
- **mcp:** Add MCP server with notes tools ([882e1b3](https://github.com/stormaaja/agent-notes/commit/882e1b37ba09ed5074ca146596c7626eb5787616))
- **cli:** Add mcp command serving notes over stdio ([8457a6f](https://github.com/stormaaja/agent-notes/commit/8457a6ff3e693358be2ff70c836c037c0e221255))
- **cli:** Add version command and --version flag ([0984395](https://github.com/stormaaja/agent-notes/commit/09843958d2238b10cf424dcdb79a6bba8754428b))

### Fixed

- **store:** Stop .notes lookup at the git root ([bbb7cd2](https://github.com/stormaaja/agent-notes/commit/bbb7cd2931219bec00410d416b37da18f6e16491))
- **store:** Normalize note body in create and update ([f344cdd](https://github.com/stormaaja/agent-notes/commit/f344cdd7c644730f335167624a50765adef74a35))
- **mcp:** Serialize tool calls ([c57fb64](https://github.com/stormaaja/agent-notes/commit/c57fb64fcfcae2518d5a3d7bc1d9bff9289689e3))
