import 'package:flutter_test/flutter_test.dart';
import 'package:theboringfloor/api/models.dart';

void main() {
  test('decodes Project', () {
    final value = Project.fromJson({
      'id': 'p',
      'dir': '/repo',
      'name': 'Repo',
      'live': true,
      'backend': 'opencode',
      'primaryId': 'main',
      'port': 8787,
      'version': '0.3.31',
      'savedAt': 1000,
      'chatCount': 8,
    });
    expect(value.name, 'Repo');
    expect(value.live, isTrue);
    expect(value.chatCount, 8);
  });
  test('decodes transcript with missing optional kind and meta', () {
    final value = TranscriptMessage.fromJson({
      'id': 'm',
      'from': 'tekton-8',
      'text': 'done',
      'at': 1000,
    });
    expect(value.kind, '');
    expect(value.meta, isNull);
    expect(value.at, 1000);
  });
  test('decodes Status', () {
    final value = Status.fromJson({
      'dir': '/repo',
      'backend': 'claude',
      'primaryId': 'p',
      'planDraftLen': 1,
      'planApprovedLen': 2,
      'chatCount': 3,
    });
    expect(value.planApprovedLen, 2);
  });
  test('decodes Busy', () {
    final value = Busy.fromJson({
      'busy': true,
      'pendingBoss': false,
      'thinking': true,
      'delegating': false,
      'questionParked': false,
    });
    expect(value.thinking, isTrue);
  });
}
