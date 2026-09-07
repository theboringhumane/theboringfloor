import 'package:flutter_test/flutter_test.dart';
import 'package:theboringfloor/models/models.dart';

void main() {
  test('decodes project and transcript', () {
    expect(
      Project.fromJson({
        'id': 'p',
        'dir': '/repo',
        'name': 'Repo',
        'live': true,
        'backend': 'x',
      }).name,
      'Repo',
    );
    expect(
      TranscriptMessage.fromJson({
        'id': 'm',
        'from': 'user',
        'text': 'ok',
        'at': 1,
      }).kind,
      '',
    );
  });
}
