import 'package:flutter/widgets.dart';

import 'app.dart';
import 'store/settings_store.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  final settings = await SettingsStore.load();
  runApp(TheBoringFloorApp(settings: settings));
}
