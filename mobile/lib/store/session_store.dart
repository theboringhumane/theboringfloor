import 'package:flutter/foundation.dart';

import '../api/gateway_client.dart';
import '../models/project.dart';
import '../models/session.dart';
import '../models/transcript.dart';

class SessionStore extends ChangeNotifier {
  SessionStore(this.client, this.project);
  final GatewayClient client;
  final Project project;
  SessionData? data;
  Object? error;
  bool loading = true;
  bool loadingOlder = false;

  List<TranscriptMessage> get messages => data?.messages ?? const [];
  bool get hasMore => data?.hasMore ?? false;
  Future<void> load() async {
    loading = true;
    error = null;
    notifyListeners();
    try {
      data = await client.session(project.id);
    } catch (e) {
      error = e;
    }
    loading = false;
    notifyListeners();
  }

  Future<void> send(String text) async {
    await client.message(project.id, text);
    await load();
  }

  /// Prepends the page immediately before the oldest retained message.
  ///
  /// The cursor is supplied directly to the gateway and is intentionally never
  /// interpreted here: it is an opaque office-owned message id.
  Future<void> loadOlder() async {
    final current = data;
    if (current == null ||
        current.messages.isEmpty ||
        !current.hasMore ||
        loadingOlder) {
      return;
    }

    loadingOlder = true;
    notifyListeners();
    try {
      final page = await client.transcript(
        project.id,
        limit: transcriptPageSize,
        before: current.messages.first.id,
      );
      data = SessionData(
        status: current.status,
        messages: [...page.messages, ...current.messages],
        hasMore: page.hasMore,
        busy: current.busy,
      );
    } on GatewayException catch (exception) {
      // A bad or expired opaque cursor cannot be repaired client-side. Stop
      // paging quietly rather than retrying it on every scroll notification.
      if (exception.statusCode == 400) {
        data = SessionData(
          status: current.status,
          messages: current.messages,
          hasMore: false,
          busy: current.busy,
        );
      }
    } finally {
      loadingOlder = false;
      notifyListeners();
    }
  }
}
