# MTG Card and Deck Manager

A Go-based microservice solution for managing Magic: The Gathering (MTG) cards, personal collections, and Commander decks. The system supports deck import, card data from Scryfall, deck analysis, and AI-generated deck descriptions.

## Architecture

- **dbinit**: Initializes the PostgreSQL schema from `app/drizzle/0000_initial.sql`.
- **scryfall**: Imports and updates card data from Scryfall dumps.
- **decks**: **Imports decks** and generates deck descriptions using OpenAI.
- **analysis**: Analyzes decks for statistics and archetype features.

## Database Schema

The schema is defined in `app/drizzle/0000_initial.sql` and includes:
- `cards`: All MTG cards (from Scryfall).
- `owned_cards`: Your personal collection.
- `decks`: Commander decks.
- `deck_cards`: Cards in a deck (mainboard, sideboard, maybeboard, commander).
- `missing_cards`: Tracks missing cards for decks.
- `deck_analysis`: Stores analysis results for decks.

## Setup Instructions

### Prerequisites
- Go 1.22 or newer
- PostgreSQL
- Scryfall card data (JSON dumps)
- OpenAI API key (for deck descriptions)

### Environment Variables
Create a `.env.local` file in the project root with the following (example):
```
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=mtgcards
OPENAI_API_KEY=your_openai_key
```

### Initialize the Database
```
cd backend/tools/dbinit
go run main.go --force
```

### Import Scryfall Card Data
Place your Scryfall card dump in `data/scryfall_dumps/` (e.g., `scryfall_cards_*.json`).
```
cd backend/tools/scryfall
go run import_cards.go
```

### Import Decks and Generate Descriptions
Place deck files as needed, then:
```
cd backend/tools/decks
go run import_decks.go   # To import decks
go run deck_describer.go # To generate AI descriptions
```

### Analyze Decks
```
cd backend/tools/analysis
go run deck_analysis.go
```

## Usage
- Add decks and cards using the import tools.
- Run analysis and description tools to enrich your deck data.
- All data is stored in PostgreSQL for easy querying and extension.

## Best Practices
- Written in idiomatic Go with proper error handling and security in mind.
- Modular microservice structure for scalability and maintainability.
- Uses Go's concurrency features where beneficial.

## Testing
- Use Go's `testing` package to test backend logic and data processing.
- Future: Add API endpoint and integration tests as needed.

---

For more details, see the code in each tool's directory and the schema in `app/drizzle/0000_initial.sql`.

