import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:theboringfloor/store/settings_store.dart';
import 'package:theboringfloor/views/settings_view.dart';

void main() {
  testWidgets('masks the token until the member explicitly reveals it', (
    tester,
  ) async {
    final store = SettingsStore(
      const GatewaySettings(
        baseUrl: 'https://gateway.example',
        token: 'secret',
      ),
    );
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(body: SettingsView(store: store)),
      ),
    );

    expect(
      tester.widget<EditableText>(find.byType(EditableText).last).obscureText,
      isTrue,
    );
    expect(find.byTooltip('Show token'), findsOneWidget);

    await tester.tap(find.byTooltip('Show token'));
    await tester.pump();
    expect(
      tester.widget<EditableText>(find.byType(EditableText).last).obscureText,
      isFalse,
    );
    expect(find.byTooltip('Hide token'), findsOneWidget);
  });

  testWidgets('saves edited gateway settings persistently', (tester) async {
    SharedPreferences.setMockInitialValues({});
    final store = await SettingsStore.load();
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(body: SettingsView(store: store)),
      ),
    );

    await tester.enterText(
      find.byType(TextField).at(0),
      'https://office.test/',
    );
    await tester.enterText(find.byType(TextField).at(1), 'new-token');
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();

    expect(store.settings.baseUrl, 'https://office.test/');
    expect(store.settings.token, 'new-token');
    final reloaded = await SettingsStore.load();
    expect(reloaded.settings.baseUrl, 'https://office.test/');
    expect(reloaded.settings.token, 'new-token');
  });
}
