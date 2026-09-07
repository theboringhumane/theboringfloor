import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:theboringfloor/api/gateway_client.dart';
import 'package:theboringfloor/app.dart';
import 'package:theboringfloor/store/settings_store.dart';

class _Client extends http.BaseClient {
  _Client(this.handler);
  final Future<http.StreamedResponse> Function(http.BaseRequest) handler;

  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) =>
      handler(request);
}

http.StreamedResponse _response(
  int status, [
  Map<String, Object?> body = const {},
]) => http.StreamedResponse(
  Stream.value(utf8.encode(jsonEncode(body))),
  status,
  headers: {'content-type': 'application/json'},
);

const _project = {
  'id': 'project-1',
  'dir': '/workspace/project',
  'name': 'Project One',
  'live': false,
  'backend': 'opencode',
  'primaryId': '',
  'port': 0,
  'version': '',
  'savedAt': 0,
  'chatCount': 0,
};

GatewayClient _gateway(
  Future<http.StreamedResponse> Function(http.BaseRequest) handler,
) => GatewayClient(
  baseUrl: 'http://gateway.test',
  token: 'token',
  httpClient: _Client(handler),
);

Widget _shell(GatewayClient client, {Duration? timeout}) => MaterialApp(
  home: AppShell(
    settings: SettingsStore(const GatewaySettings(baseUrl: '', token: '')),
    client: client,
    readinessPollInterval: const Duration(milliseconds: 1),
    readinessTimeout: timeout ?? const Duration(seconds: 1),
  ),
);

Future<void> _openPicker(WidgetTester tester) async {
  await tester.tap(find.byTooltip('New session'));
  await tester.pumpAndSettle();
  await tester.tap(find.text('Project One').last);
  await tester.pump();
}

Map<String, Object?> _status() => {
  'dir': '/workspace/project',
  'backend': 'opencode',
  'primaryId': '',
  'planDraftLen': 0,
  'planApprovedLen': 0,
  'chatCount': 0,
};

void main() {
  testWidgets('renders three destinations and swaps tab bodies', (
    tester,
  ) async {
    await tester.pumpWidget(
      _shell(
        _gateway((request) async {
          if (request.url.path == '/api/v1/projects') {
            return _response(200, {'projects': <Object>[]});
          }
          return _response(200);
        }),
      ),
    );

    expect(find.byType(NavigationDestination), findsNWidgets(3));
    expect(find.text('Space'), findsOneWidget);
    await tester.tap(find.text('Terminal'));
    await tester.pumpAndSettle();
    expect(find.text('Terminal'), findsNWidgets(2));
    await tester.tap(find.text('Settings'));
    await tester.pumpAndSettle();
    expect(find.text('Settings'), findsNWidgets(2));
  });

  testWidgets('polls an accepted office launch until it is ready', (
    tester,
  ) async {
    var polls = 0;
    await tester.pumpWidget(
      _shell(
        _gateway((request) async {
          switch (request.url.path) {
            case '/api/v1/projects':
              return _response(200, {
                'projects': [_project],
              });
            case '/api/v1/projects/project-1/start':
              return _response(202);
            case '/api/v1/projects/project-1/status':
              polls += 1;
              return polls < 2 ? _response(503) : _response(200, _status());
            case '/api/v1/projects/project-1/new':
            case '/api/v1/projects/project-1/transcript':
            case '/api/v1/projects/project-1/busy':
              return _response(200, {'messages': <Object>[]});
          }
          return _response(200);
        }),
      ),
    );

    await _openPicker(tester);
    expect(find.text('Starting office…'), findsOneWidget);
    await tester.pump(const Duration(milliseconds: 2));
    await tester.pumpAndSettle();

    expect(polls, greaterThanOrEqualTo(2));
    expect(find.byTooltip('More'), findsOneWidget);
  });

  testWidgets('shows timeout failure with a retry affordance', (tester) async {
    await tester.pumpWidget(
      _shell(
        _gateway((request) async {
          if (request.url.path == '/api/v1/projects') {
            return _response(200, {
              'projects': [_project],
            });
          }
          if (request.url.path.endsWith('/start')) {
            return _response(202);
          }
          return _response(503);
        }),
        timeout: const Duration(milliseconds: 3),
      ),
    );

    await _openPicker(tester);
    await tester.pump(const Duration(milliseconds: 5));
    await tester.pumpAndSettle();

    expect(
      find.text('The office did not become ready in time. Try again.'),
      findsOneWidget,
    );
    expect(find.text('Retry'), findsOneWidget);
  });

  testWidgets(
    'opens the session immediately when the office is already running',
    (tester) async {
      await tester.pumpWidget(
        _shell(
          _gateway((request) async {
            switch (request.url.path) {
              case '/api/v1/projects':
                return _response(200, {
                  'projects': [_project],
                });
              case '/api/v1/projects/project-1/start':
                return _response(409, {'error': 'office already running'});
              case '/api/v1/projects/project-1/status':
                return _response(200, _status());
              case '/api/v1/projects/project-1/new':
                return _response(200);
              case '/api/v1/projects/project-1/transcript':
                return _response(200, {'messages': <Object>[]});
              case '/api/v1/projects/project-1/busy':
                return _response(200, {
                  'busy': false,
                  'pendingBoss': false,
                  'thinking': false,
                  'delegating': false,
                  'questionParked': false,
                });
            }
            return _response(200);
          }),
        ),
      );

      await _openPicker(tester);
      await tester.pumpAndSettle();
      expect(find.byTooltip('More'), findsOneWidget);
    },
  );

  testWidgets('reports not found and launch failures distinctly', (
    tester,
  ) async {
    for (final failure in [
      (404, 'Project not found on the gateway.'),
      (502, 'Office could not be started: launcher failed'),
    ]) {
      await tester.pumpWidget(
        _shell(
          _gateway((request) async {
            if (request.url.path == '/api/v1/projects') {
              return _response(200, {
                'projects': [_project],
              });
            }
            return _response(failure.$1, {'error': 'launcher failed'});
          }),
        ),
      );
      await _openPicker(tester);
      await tester.pumpAndSettle();
      expect(find.text(failure.$2), findsOneWidget);
      await tester.pumpWidget(const SizedBox());
    }
  });

  testWidgets('stops readiness polling when the shell is disposed', (
    tester,
  ) async {
    await tester.pumpWidget(
      _shell(
        _gateway((request) async {
          if (request.url.path == '/api/v1/projects') {
            return _response(200, {
              'projects': [_project],
            });
          }
          return _response(request.url.path.endsWith('/start') ? 202 : 503);
        }),
      ),
    );

    await _openPicker(tester);
    await tester.pumpWidget(const SizedBox());
    await tester.pump(const Duration(seconds: 2));
    expect(tester.takeException(), isNull);
  });
}
