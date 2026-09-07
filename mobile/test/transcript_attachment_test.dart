import 'package:flutter_test/flutter_test.dart';

import 'package:theboringfloor/models/transcript.dart';

void main() {
  Map<String, dynamic> message(Map<String, dynamic> extra) => {
    'id': 'message-1',
    'from': 'user',
    'kind': 'message',
    'text': 'Hello',
    'at': 1,
    ...extra,
  };

  test('absent attachments decode to an empty list', () {
    final decoded = TranscriptMessage.fromJson(message({}));

    expect(decoded.attachments, isEmpty);
  });

  test('null attachments decode to an empty list', () {
    final decoded = TranscriptMessage.fromJson(message({'attachments': null}));

    expect(decoded.attachments, isEmpty);
  });

  test('attachments decode names and optional MIME types defensively', () {
    final decoded = TranscriptMessage.fromJson(
      message({
        'attachments': [
          {'name': 'screenshot.png', 'mime': 'image/png'},
          {'name': 'photo.jpg'},
          {'mime': 'image/png'},
          'not-a-map',
          {'name': 42, 'mime': 'image/png'},
        ],
      }),
    );

    expect(decoded.attachments, hasLength(2));
    expect(decoded.attachments[0].name, 'screenshot.png');
    expect(decoded.attachments[0].mime, 'image/png');
    expect(decoded.attachments[1].name, 'photo.jpg');
    expect(decoded.attachments[1].mime, isNull);
  });

  test('activity decodes optional string fields and ignores extra keys', () {
    final decoded = TranscriptMessage.fromJson(
      message({
        'activity': {
          'role': 'developer',
          'task': 'Run checks',
          'state': 'running',
          'future': 'ignored',
        },
      }),
    );

    expect(decoded.activity?.role, 'developer');
    expect(decoded.activity?.task, 'Run checks');
    expect(decoded.activity?.state, 'running');
  });

  test('absent, null, and non-map activity decode to null', () {
    expect(TranscriptMessage.fromJson(message({})).activity, isNull);
    expect(
      TranscriptMessage.fromJson(message({'activity': null})).activity,
      isNull,
    );
    expect(
      TranscriptMessage.fromJson(message({'activity': 'not-a-map'})).activity,
      isNull,
    );
  });

  test('non-string activity field values are ignored without throwing', () {
    final decoded = TranscriptMessage.fromJson(
      message({
        'activity': {
          'role': 4,
          'task': false,
          'state': ['running'],
        },
      }),
    );

    expect(decoded.activity, isNotNull);
    expect(decoded.activity?.role, isNull);
    expect(decoded.activity?.task, isNull);
    expect(decoded.activity?.state, isNull);
  });
}
