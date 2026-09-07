import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:theboringfloor/api/gateway_client.dart';
import 'package:theboringfloor/store/terminal_store.dart';
import 'package:theboringfloor/views/terminal_view.dart';

class _Client extends http.BaseClient {
  _Client(this.handler);
  final Future<http.StreamedResponse> Function(http.BaseRequest) handler;

  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) =>
      handler(request);
}

http.StreamedResponse _response(int status, Map<String, Object?> body) =>
    http.StreamedResponse(
      Stream.value(utf8.encode(jsonEncode(body))),
      status,
      headers: {'content-type': 'application/json'},
    );

void main() {
  testWidgets('disables command input while a command is running', (
    tester,
  ) async {
    final response = Completer<http.StreamedResponse>();
    final store = TerminalStore(
      GatewayClient(
        baseUrl: 'http://gateway.test',
        token: 'token',
        httpClient: _Client((_) => response.future),
      ),
    );
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: TerminalView(store: store, cwd: '/workspace'),
        ),
      ),
    );

    await tester.enterText(find.byType(TextField), 'pwd');
    await tester.tap(find.byTooltip('Run command'));
    await tester.pump();

    expect(tester.widget<TextField>(find.byType(TextField)).enabled, isFalse);
    expect(find.text('Running…'), findsOneWidget);

    response.complete(
      _response(200, {
        'stdout': '/workspace',
        'stderr': '',
        'exitCode': 0,
        'durationMs': 1,
        'truncated': false,
      }),
    );
    await tester.pumpAndSettle();
  });

  testWidgets('renders successful command output and command errors', (
    tester,
  ) async {
    var requestCount = 0;
    final store = TerminalStore(
      GatewayClient(
        baseUrl: 'http://gateway.test',
        token: 'token',
        httpClient: _Client((_) async {
          requestCount += 1;
          return requestCount == 1
              ? _response(200, {
                  'stdout': 'hello',
                  'stderr': '',
                  'exitCode': 0,
                  'durationMs': 4,
                  'truncated': false,
                })
              : _response(500, {'error': 'command service unavailable'});
        }),
      ),
    );
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: TerminalView(store: store, cwd: '/workspace'),
        ),
      ),
    );

    await tester.enterText(find.byType(TextField), 'echo hello');
    await tester.tap(find.byTooltip('Run command'));
    await tester.pumpAndSettle();
    expect(find.text('hello'), findsOneWidget);
    expect(find.text('exit 0 · 4ms'), findsOneWidget);

    await tester.enterText(find.byType(TextField), 'bad-command');
    await tester.tap(find.byTooltip('Run command'));
    await tester.pumpAndSettle();
    expect(find.textContaining('command service unavailable'), findsOneWidget);
  });
}
