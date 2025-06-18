-- All MTG cards
CREATE TABLE cards (
  id UUID PRIMARY KEY, -- Scryfall ID
  oracle_id UUID NOT NULL, -- Used to group card printings
  name TEXT NOT NULL, -- Printed name
  oracle_text TEXT,
  layout TEXT,
  mana_cost TEXT,
  cmc REAL, -- Converted mana cost
  type_line TEXT,
  power TEXT,
  toughness TEXT,
  loyalty TEXT,
  defense TEXT,
  colors TEXT[], -- Actual card colors
  color_identity TEXT[], -- Color identity used in deck legality
  keywords TEXT[], -- e.g., "Flying", "Lifelink"
  set_code TEXT NOT NULL, -- Scryfall set code
  collector_number TEXT NOT NULL,
  rarity TEXT,
  artist TEXT,
  image_uris JSONB, -- Partial: store normal/small/art_crop
  legalities JSONB, -- map of format -> legality
  full_data JSONB, -- Entire original JSON blob from Scryfall
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Your personal collection
CREATE TABLE owned_cards (
  id SERIAL PRIMARY KEY,
  card_id UUID REFERENCES cards(id),
  quantity INTEGER NOT NULL,
  is_foil BOOLEAN DEFAULT FALSE,
  condition TEXT, -- e.g., NM, LP, MP, etc.
  notes TEXT
);

-- Commander decks
CREATE TABLE decks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  description TEXT,
  description_gpt_model TEXT,
  commander_name TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Cards in a deck, including mainboard, sideboard, maybeboard, and commander
CREATE TABLE deck_cards (
  id SERIAL PRIMARY KEY,
  deck_id UUID REFERENCES decks(id) ON DELETE CASCADE,
  card_id UUID REFERENCES cards(id),
  quantity INTEGER NOT NULL,
  board_type TEXT NOT NULL CHECK (
    board_type IN ('commander', 'mainboard', 'sideboard', 'maybeboard')
  )
);

-- Track missing cards
CREATE TABLE missing_cards (
  id SERIAL PRIMARY KEY,
  deck_id UUID REFERENCES decks(id),
  card_id UUID REFERENCES cards(id),
  reason TEXT CHECK (reason IN ('not_owned', 'in_use_elsewhere'))
);

CREATE TABLE deck_analysis (
  deck_id UUID PRIMARY KEY REFERENCES decks(id) ON DELETE CASCADE,

  -- Mana and Curve
  average_mana_value REAL,
  highest_mana_value INTEGER,
  mana_curve JSONB,

  -- Card types and land info
  card_types JSONB,
  land_count INTEGER,
  basic_land_count INTEGER,
  nonbasic_land_count INTEGER,

  -- Colors and interaction
  color_symbols JSONB,
  draw_count INTEGER,
  single_target_removal_count INTEGER,
  mass_removal_count INTEGER,
  ramp_count INTEGER,
  counterspell_count INTEGER,
  token_count INTEGER,
  recursion_count INTEGER,

  -- Audit
  analyzed_at TIMESTAMPTZ DEFAULT NOW()
);
