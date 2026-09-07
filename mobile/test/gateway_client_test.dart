import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:theboringfloor/api/gateway_client.dart';

class FakeClient extends http.BaseClient {
  FakeClient(this.response);
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
  test('parses a gateway error body verbatim', () async {
    final client = GatewayClient(
      baseUrl: 'http://100.1.2.3:8787',
      token: 'secret',
      httpClient: FakeClient(
        http.Response(jsonEncode({'error': 'office not running'}), 409),
      ),
    );
    expect(
      () => client.projects(),
      throwsA(
        isA<GatewayException>()
            .having((error) => error.statusCode, 'status code', 409)
            .having((error) => error.message, 'message', 'office not running'),
      ),
    );
  });
}
