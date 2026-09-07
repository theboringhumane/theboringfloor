/// The terminal office's rotating status vocabulary, preserved in order from
/// `internal/panels/chat_loading.go:45-48`.
const List<String> kWorkingWords = <String>[
  'Brewing',
  'Churning',
  'Pondering',
  'Forging',
  'Crafting',
  'Scheming',
  'Sprinting',
  'Weaving',
  'Hacking',
  'Plotting',
];

/// Returns the deterministic working word for [tick].
String workingWordAt(int tick) => kWorkingWords[tick % kWorkingWords.length];
