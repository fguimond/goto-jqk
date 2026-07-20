# Considerations

- Data persistence - using in-memory for now
- Assuming that once the cards have been shuffled:
    - Decks can still be added
    - Players can still be added
- Assuming that a player is in one and only one game
- Deck creation allows creating the deck into the game using an optional GameID parameter
    - Not sure why we would have to create and add a deck in two operations since decks can't be moved. Technically an orphan deck can't be used for anything.
- Adding a few GET endpoints for troubleshooting, demos. They would not be there in a production system.
- Adding deck to the game possibilities:
    - Modify game property to manually add the deck
    - Use a *verb* endpoint like /api/v1/game/123/addDeck
    - Use PATCH endpoint with *add* operation - RFC 6902.
- Players
    - Assume a player is part of one and only one game. Once removed it can't be added another game.
    - Decided to not have a standalone players endpoint per say like `decks`, instead having a nested endpoint within the game with the following endpoints:
        - `POST /api/v1/game/123/players`
        - `DELETE /api/v1/game/123/players/456`
    - What do we do with the cards of a removed player, should we even allow to remove one?
        - For now assume cards are lost
- Shuffle
    - Use `POST /api/v1/game/123/cards/shuffle`
    - Contrary to It is not expected to be idempotent
- Deal
    - `POST /api/v1/games/{gameId}/players/{playerId}/cards   {"count": 2}`
    - cards moved from GameDeck to player
    - 409 error on not enough cards
- Last 3 endpoints for "stats" are mostly using existing data.

# Steps
- Scaffolding application + bare-bone POST/DELETE game endpoint - see README.md for features.
- POST deck
- PATCH /api/v1/games/{gameId}/decks
- Players POST/DELETE
- Shuffle GameDeck
- Deal cards to players
- Stats endpoints

# To Do
- Persistence - replace in-memory with a real database
- Shuffle uses the build-in library
- Proper user and permission management
- ~~Add logs for create, update delete operations~~
- Change default game name from Chess to Card Game