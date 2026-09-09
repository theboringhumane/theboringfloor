import 'package:flutter_test/flutter_test.dart';
import 'package:theboringfloor/models/transcript.dart';

TranscriptMessage row(
  String id,
  String from,
  String kind, {
  String? phase,
  bool pending = false,
  String? text,
}) => TranscriptMessage(
  id: id,
  from: from,
  kind: kind,
  text: text ?? id,
  at: 1,
  phase: phase,
  pending: pending,
);
void main() {
  test('only user prompts and the final primary reply reach the phone', () {
    final messages = [
      row('ask', 'user', 'user'),
      row('progress', 'boss', 'boss'),
      row('tool', 'boss', 'tool'),
      row('think', 'boss', 'think'),
      row('worker', 'frontend-1', 'message'),
      row('worker-tool', 'frontend-1', 'wtool'),
      row('notice', 'office', 'office'),
      row('debug', 'system', 'message'),
      row(
        'reply',
        'boss',
        'boss',
        text: 'Use `go test ./...`.\n```sh\ngo test ./...\n```',
      ),
      row('empty', 'boss', 'boss', text: ''),
    ];
    final visible = conversationMessages(messages);
    expect(visible.map((m) => m.id), ['ask', 'reply']);
    expect(visible.last.text, contains('```sh'));
    expect(messages.length, 10);
  });
  test('keeps real office assistant replies but not local notices', () {
    expect(
      conversationMessages([
        row('u', 'user', 'user'),
        row('notice', 'office', ''),
        row('office-answer', 'office', 'office'),
      ]).map((m) => m.id),
      ['u', 'office-answer'],
    );
  });
  test('working turns do not publish interim prose as an answer', () {
    final messages = [
      row('ask', 'user', 'user'),
      row('progress', 'boss', 'boss'),
    ];
    expect(conversationMessages(messages, working: true).map((m) => m.id), [
      'ask',
    ]);
    expect(
      conversationMessages([
        ...messages,
        row('final', 'assistant', 'message', phase: 'final'),
      ], working: true).last.id,
      'final',
    );
  });
  test(
    'keeps previous answers and attachment-only prompts; omits partials',
    () {
      final messages = [
        row('first', 'user', 'user'),
        row('answer1', 'boss', 'boss'),
        row('second', 'user', 'user'),
        row('analysis', 'assistant', 'message', phase: 'analysis'),
        row('partial', 'boss', 'boss', pending: true),
        const TranscriptMessage(
          id: 'photo',
          from: 'user',
          kind: 'user',
          text: '',
          at: 2,
          attachments: [TranscriptAttachment(name: 'design.png')],
        ),
      ];
      expect(conversationMessages(messages, working: true).map((m) => m.id), [
        'first',
        'answer1',
        'second',
        'photo',
      ]);
    },
  );
  test('a page beginning mid-turn keeps completed primary replies', () {
    expect(
      conversationMessages([
        row('old', 'assistant', 'message'),
        row('tool', 'boss', 'tool'),
        row('new', 'assistant', 'message'),
      ]).map((m) => m.id),
      ['old', 'new'],
    );
  });
}
