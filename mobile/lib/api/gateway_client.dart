import 'dart:convert';

import 'package:http/http.dart' as http;

import '../models/models.dart';
import 'endpoints.dart';

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
      (await _request('GET', GatewayEndpoints.health)) as Map<String, dynamic>;

  Future<List<Project>> projects() async {
    final data = (await _request(
      'GET',
      GatewayEndpoints.projects,
    )) as Map<String, dynamic>;
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

  Future<SessionData> session(String id) async {
    final values = await Future.wait<Object?>([
      status(id),
      transcript(id),
      _optionalBusy(id),
    ]);
    final page = values[1]! as TranscriptPage;
    return SessionData(
      status: values[0]! as Status,
      messages: page.messages,
      hasMore: page.hasMore,
      busy: values[2] as Busy?,
    );
  }

  Future<Busy?> _optionalBusy(String id) async {
    try {
      return await busy(id);
    } catch (_) {
      return null;
    }
  }

  /// Fetches the newest [limit] messages, or — when [before] is the id of the
  /// oldest message already held — the page immediately older than it. The
  /// cursor is opaque: the office resolves it, the client never interprets it.
  Future<TranscriptPage> transcript(
    String id, {
    int limit = transcriptPageSize,
    String? before,
  }) async {
    final data = await _request(
      'GET',
      '/api/v1/projects/$id/transcript',
      query: {'limit': '$limit', 'before': ?before},
    ) as Map<String, dynamic>;
    final messages = (data['messages'] as List<dynamic>)
        .cast<Map<String, dynamic>>()
        .map(TranscriptMessage.fromJson)
        .toList();
    messages.sort((left, right) => left.at.compareTo(right.at));
    return TranscriptPage(messages: messages, hasMore: data['hasMore'] == true);
  }

  Future<void> message(
    String id,
    String text, {
    List<Attachment>? attachments,
  }) {
    if (attachments == null || attachments.isEmpty) {
      return _request(
        'POST',
        '/api/v1/projects/$id/message',
        body: {'text': text},
      );
    }
    AttachmentValidator.validate(attachments);
    return _request(
      'POST',
      '/api/v1/projects/$id/message',
      body: {
        'text': text,
        'attachments': attachments.map((item) => item.toJson()).toList(),
      },
    );
  }

  Future<void> stop(String id) => _request('POST', '/api/v1/projects/$id/stop');

  Future<void> startNew(String id) =>
      _request('POST', '/api/v1/projects/$id/new');

  Future<StartOfficeResult> startOffice(String id) async {
    try {
      await _request('POST', '/api/v1/projects/$id/start');
      return const StartOfficeResult(StartOfficeOutcome.accepted);
    } on GatewayException catch (error) {
      switch (error.statusCode) {
        case 409:
          return const StartOfficeResult(StartOfficeOutcome.alreadyRunning);
        case 404:
          return const StartOfficeResult(StartOfficeOutcome.projectNotFound);
        case 502:
          return StartOfficeResult(
            StartOfficeOutcome.launchFailed,
            message: error.message,
          );
        default:
          rethrow;
      }
    }
  }

  Future<ExecResult> exec({
    required String command,
    required String cwd,
    int timeoutMs = 30000,
  }) async => ExecResult.fromJson(
    await _request(
      'POST',
      GatewayEndpoints.exec,
      body: {'command': command, 'cwd': cwd, 'timeoutMs': timeoutMs},
    ) as Map<String, dynamic>,
  );
}
