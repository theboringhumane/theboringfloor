import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:theboringfloor/components/glass.dart';
import 'package:theboringfloor/theme.dart';

void main() {
  Widget host(Widget child, ThemeData theme) => MaterialApp(
    theme: theme,
    home: Scaffold(body: Center(child: child)),
  );

  testWidgets('each glass control clips a backdrop blur', (tester) async {
    await tester.pumpWidget(
      host(
        Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            GlassSurface(child: const Text('surface')),
            GlassPill(child: const Text('pill')),
            GlassIconButton(icon: Icons.add, onPressed: () {}),
          ],
        ),
        buildLightTheme(),
      ),
    );

    expect(find.byType(ClipRRect), findsNWidgets(3));
    expect(find.byType(BackdropFilter), findsNWidgets(3));
  });

  testWidgets('intensities use distinct blur and fill values', (tester) async {
    await tester.pumpWidget(
      host(
        Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            GlassSurface(
              intensity: GlassIntensity.subtle,
              child: const SizedBox(width: 1),
            ),
            GlassSurface(
              intensity: GlassIntensity.regular,
              child: const SizedBox(width: 1),
            ),
            GlassSurface(
              intensity: GlassIntensity.strong,
              child: const SizedBox(width: 1),
            ),
          ],
        ),
        buildLightTheme(),
      ),
    );

    final filters = tester
        .widgetList<BackdropFilter>(find.byType(BackdropFilter))
        .toList();
    final fills = tester
        .widgetList<DecoratedBox>(find.byType(DecoratedBox))
        .map((box) => (box.decoration as BoxDecoration).color!.a)
        .toList();

    expect(
      filters.map((filter) => filter.filter.toString()).toSet(),
      hasLength(3),
    );
    expect(fills.toSet(), hasLength(3));
  });

  testWidgets('glass controls render under light and dark themes', (
    tester,
  ) async {
    for (final theme in [buildLightTheme(), buildDarkTheme()]) {
      await tester.pumpWidget(
        host(
          Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              GlassSurface(child: const Text('surface')),
              GlassPill(child: const Text('pill')),
              GlassIconButton(icon: Icons.search, onPressed: null),
            ],
          ),
          theme,
        ),
      );
      expect(tester.takeException(), isNull);
    }
  });

  testWidgets('pill and icon callbacks fire and disabled controls are safe', (
    tester,
  ) async {
    var pillTaps = 0;
    var iconTaps = 0;
    await tester.pumpWidget(
      host(
        Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            GlassPill(onTap: () => pillTaps++, child: const Text('tap pill')),
            GlassPill(onTap: null, child: const Text('disabled pill')),
            GlassIconButton(icon: Icons.add, onPressed: () => iconTaps++),
            GlassIconButton(icon: Icons.remove, onPressed: null),
          ],
        ),
        buildLightTheme(),
      ),
    );

    await tester.tap(find.text('tap pill'));
    await tester.tap(find.byIcon(Icons.add));
    await tester.tap(find.text('disabled pill'));
    await tester.tap(find.byIcon(Icons.remove));

    expect(pillTaps, 1);
    expect(iconTaps, 1);
    expect(tester.takeException(), isNull);
  });
}
