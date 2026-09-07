import 'dart:convert';

import 'package:http/http.dart' as http;

import 'models.dart';

class GatewayException implements Exception {
  const GatewayException(this.statusCode, this.message);

  final int statusCode;
  final String message;

  @override
  String toString() => 'GatewayException($statusCode): $message';
}

class GatewayClient {
  GatewayClient({
    required this.baseUrl,
    required this.token,
    http.Client? httpClient,
  }) : _http = httpClient ?? http.Client();

  final String baseUrl;
  final String token;
  final http.Client _http;

  Uri _uri(String path, [Map<String, String>? query]) =>
      Uri.parse('${baseUrl.replaceFirst(RegExp(r'/$'), '')}$path')
          .replace(queryParameters: query);

  Map<String, String> get _headers => {
    'Authorization': 'Bearer $token',
    'Content-Type': 'application/json',
  };

  Future<dynamic> _request(
    String method,
    String path, {
    Map<String, String>? query,
    Object? body,
  }) async {
    final request = http.Request(method, _uri(path, query))
      ..headers.addAll(_headers);
    if (body != null) request.body = jsonEncode(body);
    final streamed = await _http.send(request);
    final response = await http.Response.fromStream(streamed);
    dynamic decoded;
    if (response.body.isNotEmpty) {
      try {
        decoded = jsonDecode(response.body);
      } on FormatException {
        decoded = null;
      }
    }
    if (response.statusCode < 200 || response.statusCode >= 300) {
      final message =
          decoded is Map<String, dynamic> && decoded['error'] is String
          ? decoded['error'] as String
          : response.body;
      throw GatewayException(response.statusCode, message);
    }
    return decoded;
  }

  Future<Map<String, dynamic>> health() async =>
      (await _request('GET', '/api/v1/health')) as Map<String, dynamic>;

  Future<List<Project>> projects() async {
    final data =
        (await _request('GET', '/api/v1/projects')) as Map<String, dynamic>;
    return (data['projects'] as List<dynamic>)
        .cast<Map<String, dynamic>>()
        .map(Project.fromJson)
        .toList();
  }

  Future<Project> project(String id) async => Project.fromJson(
    (await _request('GET', '/api/v1/projects/$id')) as Map<String, dynamic>,
  );

  Future<Status> status(String id) async => Status.fromJson(
    (await _request('GET', '/api/v1/projects/$id/status'))
        as Map<String, dynamic>,
  );

  Future<Busy> busy(String id) async => Busy.fromJson(
    (await _request('GET', '/api/v1/projects/$id/busy'))
        as Map<String, dynamic>,
  );

  Future<List<TranscriptMessage>> transcript(
    String id, {
    int limit = 250,
  }) async {
    final data = await _request(
      'GET',
      '/api/v1/projects/$id/transcript',
      query: {'limit': '$limit'},
    ) as Map<String, dynamic>;
    final messages = (data['messages'] as List<dynamic>)
        .cast<Map<String, dynamic>>()
        .map(TranscriptMessage.fromJson)
        .toList();
    messages.sort((left, right) => left.at.compareTo(right.at));
    return messages;
  }

  Future<void> message(String id, String text) =>
      _request('POST', '/api/v1/projects/$id/message', body: {'text': text});

  Future<void> stop(String id) => _request('POST', '/api/v1/projects/$id/stop');

  Future<void> startNew(String id) =>
      _request('POST', '/api/v1/projects/$id/new');
}
