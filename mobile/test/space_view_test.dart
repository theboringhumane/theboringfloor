import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:theboringfloor/api/gateway_client.dart';
import 'package:theboringfloor/models/project.dart';
import 'package:theboringfloor/store/projects_store.dart';
import 'package:theboringfloor/utils/typography.dart';
import 'package:theboringfloor/views/space_view.dart';

class _ProjectsClient extends http.BaseClient {
  _ProjectsClient(this.projects, {this.responseCompleter});

  final List<Project> projects;
  final Completer<http.Response>? responseCompleter;
  int requests = 0;

  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    requests++;
    final response = responseCompleter != null
        ? await responseCompleter!.future
        : http.Response(_projectsResponse(projects), 200);
    return http.StreamedResponse(
      Stream.value(response.bodyBytes),
      response.statusCode,
      headers: response.headers,
      request: request,
    );
  }
}

String _projectsResponse(List<Project> projects) => jsonEncode({
  'projects': projects
      .map(
        (project) => {
          'id': project.id,
          'dir': project.dir,
          'name': project.name,
          'live': project.live,
          'backend': project.backend,
          'primaryId': project.primaryId,
          'port': project.port,
          'version': project.version,
          'savedAt': project.savedAt,
          'chatCount': project.chatCount,
        },
      )
      .toList(),
});

final _running = Project(
  id: 'running',
  dir: '/work/alpha',
  name: 'Alpha Office',
  live: true,
  backend: 'opencode',
  primaryId: '',
  port: 0,
  version: '',
  savedAt: 0,
  chatCount: 2,
);

final _stopped = Project(
  id: 'stopped',
  dir: '/work/beta',
  name: 'Beta Office',
  live: false,
  backend: 'claude',
  primaryId: '',
  port: 0,
  version: '',
  savedAt: 0,
  chatCount: 1,
);

ProjectsStore _store(_ProjectsClient httpClient) => ProjectsStore(
  GatewayClient(
    baseUrl: 'http://gateway.test',
    token: 'test-token',
    httpClient: httpClient,
  ),
);

Future<void> _pumpSpace(
  WidgetTester tester,
  ProjectsStore store, {
  ValueChanged<Project>? onOpen,
}) async {
  await tester.pumpWidget(
    MaterialApp(
      home: Scaffold(
        body: SpaceView(store: store, onOpen: onOpen ?? (_) {}),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

Future<void> _openSearch(WidgetTester tester) async {
  await tester.tap(find.byTooltip('Search projects'));
  await tester.pumpAndSettle();
}

Future<void> _selectFilter(WidgetTester tester, String label) async {
  await tester.tap(find.byTooltip('Filter projects'));
  await tester.pumpAndSettle();
  await tester.tap(find.widgetWithText(FilterChip, label));
  await tester.pumpAndSettle();
}

void main() {
  setUpAll(disableRuntimeFontFetching);

  group('SpaceView', () {
    testWidgets('renders the home hero title in Playfair Display', (
      tester,
    ) async {
      await _pumpSpace(tester, _store(_ProjectsClient([_running])));

      final title = tester.widget<Text>(find.text('Floors'));
      expect(title.style?.fontFamily, 'Playfair Display');
    });

    testWidgets('filters live by project name and directory', (tester) async {
      await _pumpSpace(tester, _store(_ProjectsClient([_running, _stopped])));

      await _openSearch(tester);
      await tester.enterText(find.byType(TextField), 'alpha');
      await tester.pump();
      expect(find.text('Alpha Office'), findsOneWidget);
      expect(find.text('Beta Office'), findsNothing);

      await tester.enterText(find.byType(TextField), '/work/beta');
      await tester.pump();
      expect(find.text('Alpha Office'), findsNothing);
      expect(find.text('Beta Office'), findsOneWidget);
    });

    testWidgets('uses All by default and each liveness filter works', (
      tester,
    ) async {
      await _pumpSpace(tester, _store(_ProjectsClient([_running, _stopped])));

      expect(find.text('Alpha Office'), findsOneWidget);
      expect(find.text('Beta Office'), findsOneWidget);
      expect(find.text('RUNNING'), findsOneWidget);
      expect(find.text('STOPPED'), findsOneWidget);

      await _selectFilter(tester, 'Running');
      expect(find.text('Alpha Office'), findsOneWidget);
      expect(find.text('Beta Office'), findsNothing);
      expect(find.text('RUNNING'), findsOneWidget);
      expect(find.text('STOPPED'), findsNothing);

      await _selectFilter(tester, 'Stopped');
      expect(find.text('Alpha Office'), findsNothing);
      expect(find.text('Beta Office'), findsOneWidget);
      expect(find.text('RUNNING'), findsNothing);
      expect(find.text('STOPPED'), findsOneWidget);

      await _selectFilter(tester, 'All');
      expect(find.text('Alpha Office'), findsOneWidget);
      expect(find.text('Beta Office'), findsOneWidget);
    });

    testWidgets('composes search and chips and shows the empty state', (
      tester,
    ) async {
      await _pumpSpace(tester, _store(_ProjectsClient([_running, _stopped])));

      await _selectFilter(tester, 'Stopped');
      await _openSearch(tester);
      await tester.enterText(find.byType(TextField), 'alpha');
      await tester.pump();

      expect(find.text('No projects match your filters.'), findsOneWidget);
      expect(find.byType(CircularProgressIndicator), findsNothing);
    });

    testWidgets('shows distinct status dots and opens the tapped project', (
      tester,
    ) async {
      Project? opened;
      await _pumpSpace(
        tester,
        _store(_ProjectsClient([_running, _stopped])),
        onOpen: (project) => opened = project,
      );

      expect(find.text('LIVE · /work/alpha'), findsOneWidget);
      expect(find.text('STOPPED · /work/beta'), findsOneWidget);
      final runningDot = tester.widget<Container>(
        find.byKey(const Key('status-dot-running')),
      );
      final stoppedDot = tester.widget<Container>(
        find.byKey(const Key('status-dot-stopped')),
      );
      expect(
        (runningDot.decoration! as BoxDecoration).color,
        isNot((stoppedDot.decoration! as BoxDecoration).color),
      );
      await tester.tap(find.text('Beta Office'));
      expect(opened?.id, _stopped.id);
    });

    testWidgets('pull-to-refresh loads projects again', (tester) async {
      final httpClient = _ProjectsClient([_running]);
      await _pumpSpace(tester, _store(httpClient));
      expect(httpClient.requests, 1);

      await tester.drag(find.byType(ListView), const Offset(0, 300));
      await tester.pump();
      await tester.pump(const Duration(seconds: 1));

      expect(httpClient.requests, 2);
    });

    testWidgets('shows loading progress without an empty-state flash', (
      tester,
    ) async {
      final response = Completer<http.Response>();
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SpaceView(
              store: _store(_ProjectsClient([], responseCompleter: response)),
              onOpen: (_) {},
            ),
          ),
        ),
      );
      await tester.pump();

      expect(find.byType(CircularProgressIndicator), findsOneWidget);
      expect(find.text('No projects match your filters.'), findsNothing);
      response.complete(http.Response(_projectsResponse([]), 200));
    });
  });
}
