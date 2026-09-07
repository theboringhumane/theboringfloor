import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:theboringfloor/api/gateway_client.dart';
import 'package:theboringfloor/models/project.dart';
import 'package:theboringfloor/store/projects_store.dart';
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

void main() {
  group('SpaceView', () {
    testWidgets('filters live by project name and directory', (tester) async {
      await _pumpSpace(tester, _store(_ProjectsClient([_running, _stopped])));

      await tester.enterText(find.byType(TextField), 'alpha');
      await tester.pump();
      expect(find.text('Alpha Office'), findsOneWidget);
      expect(find.text('Beta Office'), findsNothing);

      await tester.enterText(find.byType(TextField), '/work/beta');
      await tester.pump();
      expect(find.text('Alpha Office'), findsNothing);
      expect(find.text('Beta Office'), findsOneWidget);
    });

    testWidgets('uses All by default and each liveness chip filters', (
      tester,
    ) async {
      await _pumpSpace(tester, _store(_ProjectsClient([_running, _stopped])));

      expect(find.text('Alpha Office'), findsOneWidget);
      expect(find.text('Beta Office'), findsOneWidget);
      expect(
        tester
            .widget<FilterChip>(find.widgetWithText(FilterChip, 'All'))
            .selected,
        isTrue,
      );

      await tester.tap(find.text('Running'));
      await tester.pump();
      expect(find.text('Alpha Office'), findsOneWidget);
      expect(find.text('Beta Office'), findsNothing);
      expect(
        tester
            .widget<FilterChip>(find.widgetWithText(FilterChip, 'Running'))
            .selected,
        isTrue,
      );
      expect(
        tester
            .widget<FilterChip>(find.widgetWithText(FilterChip, 'All'))
            .selected,
        isFalse,
      );

      await tester.tap(find.text('Stopped'));
      await tester.pump();
      expect(find.text('Alpha Office'), findsNothing);
      expect(find.text('Beta Office'), findsOneWidget);
      expect(
        tester
            .widget<FilterChip>(find.widgetWithText(FilterChip, 'Stopped'))
            .selected,
        isTrue,
      );
      expect(
        tester
            .widget<FilterChip>(find.widgetWithText(FilterChip, 'Running'))
            .selected,
        isFalse,
      );

      await tester.tap(find.text('All'));
      await tester.pump();
      expect(find.text('Alpha Office'), findsOneWidget);
      expect(find.text('Beta Office'), findsOneWidget);
    });

    testWidgets('composes search and chips and shows the empty state', (
      tester,
    ) async {
      await _pumpSpace(tester, _store(_ProjectsClient([_running, _stopped])));

      await tester.tap(find.text('Stopped'));
      await tester.enterText(find.byType(TextField), 'alpha');
      await tester.pump();

      expect(find.text('No projects match your filters.'), findsOneWidget);
      expect(find.byType(CircularProgressIndicator), findsNothing);
    });

    testWidgets('shows liveness badges and opens the tapped project', (
      tester,
    ) async {
      Project? opened;
      await _pumpSpace(
        tester,
        _store(_ProjectsClient([_running, _stopped])),
        onOpen: (project) => opened = project,
      );

      expect(find.text('LIVE'), findsOneWidget);
      expect(find.text('STOPPED'), findsOneWidget);
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
