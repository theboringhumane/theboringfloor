import 'package:flutter/foundation.dart';

import '../api/gateway_client.dart';
import '../models/project.dart';
import '../models/session.dart';
import '../models/transcript.dart';
import '../models/attachment.dart';

/// How often an open transcript asks the gateway for its newest page.
const sessionRefreshInterval = Duration(seconds: 3);

class SessionStore extends ChangeNotifier {
  SessionStore(this.client, this.project);
  final GatewayClient client;
  final Project project;
  SessionData? data;
  Object? error;
  bool loading = true;
  bool loadingOlder = false;
  bool sending = false;
  String? sendError;
  bool _hasPagedOlder = false;
  bool _refreshing = false;
  int _pollGeneration = 0;
  int _newestMessageGeneration = 0;

  List<TranscriptMessage> get messages => data?.messages ?? const [];
  bool get hasMore => data?.hasMore ?? false;
  int get pollGeneration => _pollGeneration;

  /// Advances only when a refresh brings a previously unseen newest message.
  ///
  /// Views use this to distinguish live arrivals from older-page merges,
  /// without coupling transcript rendering to pagination state.
  int get newestMessageGeneration => _newestMessageGeneration;
  bool get isWorking => data?.busy?.busy == true;
  Future<void> load() async {
    _pollGeneration += 1;
    loading = true;
    error = null;
    notifyListeners();
    try {
      data = await client.session(project.id);
      _hasPagedOlder = false;
    } catch (e) {
      error = e;
    }
    loading = false;
    notifyListeners();
  }

  /// Silently merges the gateway's newest page into the retained transcript.
  ///
  /// This deliberately avoids [loading] and preserves paged-in history, so a
  /// periodic update cannot replace the transcript or move its scroll offset.
  Future<bool> refresh() async {
    // A newest-page request and an older-page request cannot safely race: the
    // latter's opaque cursor is based on the currently retained oldest item.
    if (_refreshing || loadingOlder) return true;

    _refreshing = true;
    try {
      final latest = await client.session(project.id);
      final current = data;
      if (current == null) {
        data = latest;
        notifyListeners();
        return true;
      }

      final mergedMessages = _mergeMessages(current.messages, latest.messages);
      final currentIds = current.messages.map((message) => message.id).toSet();
      final hasNewMessages = latest.messages.any(
        (message) => !currentIds.contains(message.id),
      );
      final hasMore = _hasPagedOlder ? current.hasMore : latest.hasMore;
      final changed =
          !_sameMessages(current.messages, mergedMessages) ||
          !_sameStatus(current.status, latest.status) ||
          !_sameBusy(current.busy, latest.busy) ||
          current.hasMore != hasMore;
      if (!changed) return true;

      data = SessionData(
        status: latest.status,
        messages: mergedMessages,
        hasMore: hasMore,
        busy: latest.busy,
      );
      if (hasNewMessages) {
        _newestMessageGeneration += 1;
      }
      notifyListeners();
      return true;
    } catch (_) {
      // A transient polling failure must not obscure an already usable chat.
      return false;
    } finally {
      _refreshing = false;
    }
  }

  Future<bool> send(
    String text, {
    List<Attachment> attachments = const [],
  }) async {
    if (sending) return false;
    sending = true;
    sendError = null;
    notifyListeners();
    try {
      await client.message(project.id, text, attachments: attachments);
      await load();
      return true;
    } on AttachmentValidationException catch (error) {
      sendError = error.message;
      return false;
    } on GatewayException catch (error) {
      sendError = switch (error.statusCode) {
        400 => error.message,
        413 => 'Image too large to send.',
        _ => 'Couldn’t send message. Try again.',
      };
      return false;
    } catch (_) {
      sendError = 'Couldn’t send message. Try again.';
      return false;
    } finally {
      sending = false;
      notifyListeners();
    }
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
        loadingOlder ||
        _refreshing) {
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
        // The server's limit counts user messages, so a page can overlap the
        // retained history and contain arbitrarily many activity entries.
        // Merge by id and timestamp rather than assuming a fixed page shape.
        messages: _mergeMessages(current.messages, page.messages),
        hasMore: page.hasMore,
        busy: current.busy,
      );
      _hasPagedOlder = true;
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

  List<TranscriptMessage> _mergeMessages(
    List<TranscriptMessage> retained,
    List<TranscriptMessage> latest,
  ) {
    final byId = <String, TranscriptMessage>{
      for (final message in retained) message.id: message,
      for (final message in latest) message.id: message,
    };
    final messages = byId.values.toList()
      ..sort((left, right) => left.at.compareTo(right.at));
    return messages;
  }

  bool _sameMessages(
    List<TranscriptMessage> left,
    List<TranscriptMessage> right,
  ) {
    if (left.length != right.length) return false;
    for (var index = 0; index < left.length; index += 1) {
      final a = left[index];
      final b = right[index];
      if (a.id != b.id ||
          a.from != b.from ||
          a.kind != b.kind ||
          a.text != b.text ||
          a.at != b.at ||
          a.meta != b.meta) {
        return false;
      }
    }
    return true;
  }

  bool _sameStatus(Status left, Status right) =>
      left.dir == right.dir &&
      left.backend == right.backend &&
      left.primaryId == right.primaryId &&
      left.planDraftLen == right.planDraftLen &&
      left.planApprovedLen == right.planApprovedLen &&
      left.chatCount == right.chatCount;

  bool _sameBusy(Busy? left, Busy? right) =>
      left?.busy == right?.busy &&
      left?.pendingBoss == right?.pendingBoss &&
      left?.thinking == right?.thinking &&
      left?.delegating == right?.delegating &&
      left?.questionParked == right?.questionParked;
}
