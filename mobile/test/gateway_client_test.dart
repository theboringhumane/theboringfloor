import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:theboringfloor/api/gateway_client.dart';

class Fake extends http.BaseClient {
  Fake(this.response);
  final http.Response response;
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async =>
      http.StreamedResponse(
        Stream.value(response.bodyBytes),
        response.statusCode,
        headers: response.headers,
        request: request,
      );
}

void main() {
  test('exec sends documented request and parses response', () async {
    final client = GatewayClient(
      baseUrl: 'http://gateway.test',
      token: 'secret',
      httpClient: Fake(
        http.Response(
          jsonEncode({
            'stdout': 'ok',
            'stderr': '',
            'exitCode': 0,
            'durationMs': 2,
            'truncated': false,
          }),
          200,
        ),
      ),
    );
    final result = await client.exec(
      command: 'pwd',
      cwd: '/repo',
      timeoutMs: 100,
    );
    expect(result.stdout, 'ok');
  });
  test('gateway error preserves status', () async {
    final client = GatewayClient(
      baseUrl: 'http://x',
      token: 't',
      httpClient: Fake(http.Response(jsonEncode({'error': 'disabled'}), 403)),
    );
    expect(
      () => client.exec(command: 'x', cwd: '/'),
      throwsA(isA<GatewayException>()),
    );
  });
}
