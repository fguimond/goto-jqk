# Changelog

## 0.1.0 (2026-07-20)


### ⚠ BREAKING CHANGES

* move deck cards into a game deck on assignment ([#6](https://github.com/fguimond/goto-jqk/issues/6))
* pluralize REST resource paths ([#3](https://github.com/fguimond/goto-jqk/issues/3))

### Features

* add deck resource with POST /api/v1/deck ([13a0445](https://github.com/fguimond/goto-jqk/commit/13a044526638cf57766742fa2568728ecf9d38ba))
* add decks to a game via RFC 6902 patch ([71dcad7](https://github.com/fguimond/goto-jqk/commit/71dcad7338b4b3dffe1b2f81aff0f086d40ee9b2))
* add GET /api/v1/games and GET /api/v1/decks ([#4](https://github.com/fguimond/goto-jqk/issues/4)) ([c36cf4f](https://github.com/fguimond/goto-jqk/commit/c36cf4fe185aab0fec04e8a4e929dce952d1b1f4))
* add GET /api/v1/games/{gameId}/cards ([#7](https://github.com/fguimond/goto-jqk/issues/7)) ([55e95a7](https://github.com/fguimond/goto-jqk/commit/55e95a782bbbc18b1236b1792db28300e7433f13))
* add GET /api/v1/games/{gameId}/cards/counts ([#13](https://github.com/fguimond/goto-jqk/issues/13)) ([c82a869](https://github.com/fguimond/goto-jqk/commit/c82a869fede164be9e60771e3ea4b2203101cdfa))
* add GET /api/v1/games/{gameId}/cards/suits ([#12](https://github.com/fguimond/goto-jqk/issues/12)) ([4cef75e](https://github.com/fguimond/goto-jqk/commit/4cef75e44bf75fa47287ad53c3664540fc444029))
* add GET /api/v1/games/{gameId}/leaders ([#11](https://github.com/fguimond/goto-jqk/issues/11)) ([a363dd1](https://github.com/fguimond/goto-jqk/commit/a363dd1034757ae011fded29af7eed49b8de7a91))
* add GET /api/v1/games/{gameId}/players/{playerId}/cards ([#10](https://github.com/fguimond/goto-jqk/issues/10)) ([1493479](https://github.com/fguimond/goto-jqk/commit/1493479daa8e0a7bc1821a79c3a53640e677db71))
* add players to a game ([#5](https://github.com/fguimond/goto-jqk/issues/5)) ([a5eef70](https://github.com/fguimond/goto-jqk/commit/a5eef70500914f6cee862c1a205966a7ae3af3c5))
* add POST /api/v1/games/{gameId}/cards/shuffle ([#8](https://github.com/fguimond/goto-jqk/issues/8)) ([f4ad442](https://github.com/fguimond/goto-jqk/commit/f4ad44285f8a4a1a7cd3efd3be71b85a3a8bcb42))
* add POST /api/v1/games/{gameId}/players/{playerId}/cards ([#9](https://github.com/fguimond/goto-jqk/issues/9)) ([b1a9ca4](https://github.com/fguimond/goto-jqk/commit/b1a9ca44dbc27196d4012aa75b4fd18e5196a2e1))
* log create, update and delete operations ([#17](https://github.com/fguimond/goto-jqk/issues/17)) ([d18d607](https://github.com/fguimond/goto-jqk/commit/d18d60773f0af40fc7eb462b2bc4f3bfe7990f05))
* move deck cards into a game deck on assignment ([#6](https://github.com/fguimond/goto-jqk/issues/6)) ([d288827](https://github.com/fguimond/goto-jqk/commit/d288827c6c45cc394e0510a9bf29cdc2e4f442de))
* pluralize REST resource paths ([#3](https://github.com/fguimond/goto-jqk/issues/3)) ([096992f](https://github.com/fguimond/goto-jqk/commit/096992f4d364326583f198bd18d762d9c335d08c))
* scaffold goto-jqk REST API ([03166ef](https://github.com/fguimond/goto-jqk/commit/03166efa08c3e50c8fc378d441cd861ebb2f89d0))
