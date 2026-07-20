This document highlights the Claude Code prompts that were use when completing this deck of card game.


---
Create a project scaffolding for a Go application implementing a rest api.
The application should support command line arguments using cobra/viper similar to kubectl. Initial commands include `run`, `version`, `completion` and `help`.
The application should have JSON structured logs using log/slog.
Git support and the module being github.com/fguimond/goto-jqk.
Start with the POST and DELETE game endpoints in `/api/v1/game`.
A game only has a name property for now.
Ids should be UUID v4.
The api should support openapi and include a `/docs` endpoint.
The api should include a `/healthz` endpoint.
Add github actions for linting (golangci-lint) and building checks on a new push.
- use latest versions for each github action
  Add release-please github action which builds and attaches binaries for windows, linux and darwin, each for amd64 and arm64. The version number should be injected at that time
  The project should have the following layers:
* HTTP handler
* services
* store - for now let's assume in-memory storage
  Include a `README.md` file with proper information.
  Instead of making assumptions please ask questions if you need clarifications.

---
the way you generated interfaces is not idiomatic to go. they should be defined where they are used, not where the implementation is defined. for example @internal/service/game.go should define the store interface instead of it being defined in @internal/store/store.go. please review all interface usage in the project and them the idiomatic way.

	self: add store/memory package

---
i want to add an operation to add a deck to a game. create the PATCH /api/v1/game/{gameId}/decks following the patch rfc 6902 which supports exclusively the `add` operation in the body with a list of decks to be added to the game. no other operation like `test` or `remove`  should be supported. the add body should be this:
  ```
  [
      { "op": "add", "path": "/-", "value": "deck-uuid" }
  ]
  ``` 

---
make endpoints plural. for ex. change `/api/v1/game` to `/api/v1/games`

---
**For troubleshooting/demo purposes**
create GET /api/v1/games and GET /api/v1/decks

---
Add a `POST /api/v1/games/{gameId}/players` endpoint to create a player. A player has the following properties:
- ID - generated uuid
- Name - string - required
- GameID - uuid - optional
- cards - empty when created
  The game should also have a `players` property which is empty when the game is created.
  Also create a `DELETE /api/v1/games/{gameId}/players/{playerId}` endpoint to remove a player from a game.

---
First thing, let's add a GameDeck property to the Game. When a deck is added to a game the deck's card should be moved out of the deck and appended into the game's GameDeck. The game endpoint should list the number of cards remaining in the GameDeck.

---
Secondly add a `GET /api/v1/games/{gameId}/cards` that returns the list of cards in the order they are.

---
i want to add a way to shuffle cards from a game's GameDeck. what approach do you recommend i use as far as rest endpoint signature. how does that fit in regards to the rest philosophy?

---
Add a `GET /api/v1/games/{gameId}/players/{playerId}/cards` endpoint that returns the list of cards for that player

---
Add a `GET /api/v1/games/{gameId}/leaders` endpoint that list the players of a game along with the total added value of all the cards each player holds. Use face value only. Card values are as follow: ace=1, 2-10=same value, jack=11, queen=12, king=13. The list returned should be in descending order.

---
Add a `GET /api/v1/games/{gameId}/cards/suits` that shows how many cards per suit are left in the GameDeck of a game. Ex. `{"spade":4, "heart": 10}`. Return format should be `[{"suit":"heart", "count": 5}, {"suit":"spade", "remainging":12}`

---
Add an endpoint that counts each card remaining in the GameDeck, sorted by suit (alphabetical, ascending order) and card value in descending order (value as already defined - ace=1, 2-10, jack=11, queen=12, king=13) from high value to low value. please confirm with me what the endpoint should look like.

---
add detailed info logs for create, update and delete operations.

instead just having msg="game created", also have two other properties, `entity` and `operation`. something like `{"entity"="game", "operation":"create}, ...}`

---