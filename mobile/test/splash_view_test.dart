import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:theboringfloor/app.dart';
import 'package:theboringfloor/store/projects_store.dart';
import 'package:theboringfloor/store/settings_store.dart';
import 'package:theboringfloor/theme.dart';
import 'package:theboringfloor/views/splash_view.dart';

AppBootstrap _bootstrap() {
  final settings = SettingsStore(const GatewaySettings(baseUrl: '', token: ''));
  return AppBootstrap(
    settings: settings,
    projects: ProjectsStore(settings.client()),
  );
}

void main() {
  testWidgets('renders the logo and progress indicator in both themes', (
    tester,
  ) async {
    for (final theme in [buildLightTheme(), buildDarkTheme()]) {
      await tester.pumpWidget(
        MaterialApp(theme: theme, home: const SplashView()),
      );

      expect(find.byType(Image), findsOneWidget);
      expect(find.byType(CircularProgressIndicator), findsOneWidget);
      expect(find.byType(ColoredBox), findsAtLeastNWidgets(1));
      await tester.pumpWidget(const SizedBox());
    }
  });

  testWidgets('shows splash while bootstrapping then replaces it with shell', (
    tester,
  ) async {
    final completer = Completer<AppBootstrap>();
    await tester.pumpWidget(
      TheBoringFloorApp(bootstrap: () => completer.future),
    );

    expect(find.byType(SplashView), findsOneWidget);
    expect(find.byType(NavigationBar), findsNothing);

    completer.complete(_bootstrap());
    await tester.pumpAndSettle();

    expect(find.byType(SplashView), findsNothing);
    expect(find.byType(NavigationDestination), findsNWidgets(4));
  });

  testWidgets('shows a failed bootstrap and retries successfully', (
    tester,
  ) async {
    var attempts = 0;
    await tester.pumpWidget(
      TheBoringFloorApp(
        bootstrap: () async {
          attempts += 1;
          if (attempts == 1) {
            throw StateError('offline');
          }
          return _bootstrap();
        },
      ),
    );
    await tester.pumpAndSettle();

    expect(
      find.text(
        'We could not finish starting the app. Check your connection and try again.',
      ),
      findsOneWidget,
    );
    await tester.tap(find.text('Retry'));
    await tester.pumpAndSettle();

    expect(attempts, 2);
    expect(find.byType(NavigationDestination), findsNWidgets(4));
  });

  testWidgets('turns a slow bootstrap into a retryable timeout', (
    tester,
  ) async {
    final completer = Completer<AppBootstrap>();
    await tester.pumpWidget(
      TheBoringFloorApp(
        bootstrap: () => completer.future,
        bootstrapTimeout: const Duration(milliseconds: 10),
      ),
    );
    await tester.pump(const Duration(milliseconds: 11));

    expect(
      find.text('The app took too long to start. Try again.'),
      findsOneWidget,
    );
    expect(find.text('Retry'), findsOneWidget);
  });

  testWidgets('ignores a bootstrap completion after disposal', (tester) async {
    final completer = Completer<AppBootstrap>();
    await tester.pumpWidget(
      TheBoringFloorApp(bootstrap: () => completer.future),
    );
    await tester.pumpWidget(const SizedBox());
    completer.complete(_bootstrap());
    await tester.pump();

    expect(tester.takeException(), isNull);
  });
}
