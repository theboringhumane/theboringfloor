import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:theboringfloor/utils/typography.dart';

void main() {
  const families = <String>[
    'Inter',
    'Space Grotesk',
    'Playfair Display',
    'JetBrains Mono',
  ];
  const assetsByFamily = <String, List<String>>{
    'Inter': <String>[
      'assets/fonts/Inter-Regular.ttf',
      'assets/fonts/Inter-Medium.ttf',
      'assets/fonts/Inter-Bold.ttf',
    ],
    'Space Grotesk': <String>[
      'assets/fonts/SpaceGrotesk-Regular.ttf',
      'assets/fonts/SpaceGrotesk-Medium.ttf',
      'assets/fonts/SpaceGrotesk-Bold.ttf',
    ],
    'Playfair Display': <String>[
      'assets/fonts/PlayfairDisplay-Regular.ttf',
      'assets/fonts/PlayfairDisplay-Bold.ttf',
    ],
    'JetBrains Mono': <String>[
      'assets/fonts/JetBrainsMono-Regular.ttf',
      'assets/fonts/JetBrainsMono-Medium.ttf',
      'assets/fonts/JetBrainsMono-Bold.ttf',
    ],
  };

  setUpAll(disableRuntimeFontFetching);

  testWidgets('bundled font families resolve from declared asset fonts', (
    tester,
  ) async {
    final manifest = await AssetManifest.loadFromAssetBundle(rootBundle);
    final manifestAssets = manifest.listAssets();
    final fontManifest = jsonDecode(
      await rootBundle.loadString('FontManifest.json'),
    ) as List<dynamic>;
    for (final entry in assetsByFamily.entries) {
      for (final asset in entry.value) {
        expect(manifestAssets, contains(asset));
      }
      expect(
        fontManifest,
        contains(
          predicate<dynamic>(
            (font) =>
                font['family'] == entry.key &&
                (font['fonts'] as List<dynamic>)
                    .map((asset) => asset['asset'] as String)
                    .toSet()
                    .containsAll(entry.value),
            'declares ${entry.key} with every bundled asset',
          ),
        ),
      );
    }

    late BuildContext context;
    await tester.pumpWidget(
      MaterialApp(
        home: Builder(
          builder: (value) {
            context = value;
            return Column(
              children: <Widget>[
                Text('body', style: AppFonts.body(context)),
                Text('heading', style: AppFonts.heading(context)),
                Text('serif', style: AppFonts.serif(context)),
                Text('mono', style: AppFonts.mono(context)),
              ],
            );
          },
        ),
      ),
    );

    for (var index = 0; index < families.length; index++) {
      final text = tester.widget<Text>(
        find.text(<String>['body', 'heading', 'serif', 'mono'][index]),
      );
      expect(text.style?.fontFamily, families[index]);
      expect(text.style?.fontFamily, isNot('Roboto'));
    }
  });
}
