import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:theboringfloor/components/app_nav_bar.dart';

void main() {
  testWidgets('nav bar renders three destinations and switches tabs', (
    t,
  ) async {
    var index = 0;
    await t.pumpWidget(
      MaterialApp(
        home: StatefulBuilder(
          builder: (context, setState) => Scaffold(
            body: Text(['Space view', 'Terminal view', 'Settings view'][index]),
            bottomNavigationBar: AppNavBar(
              index: index,
              onChanged: (v) => setState(() => index = v),
            ),
          ),
        ),
      ),
    );
    expect(find.text('Floors'), findsOneWidget);
    expect(find.text('Terminal'), findsOneWidget);
    expect(find.text('Settings'), findsOneWidget);
    await t.tap(find.text('Terminal'));
    await t.pump();
    expect(find.text('Terminal view'), findsOneWidget);
  });
}
