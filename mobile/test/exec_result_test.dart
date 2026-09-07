import 'package:flutter_test/flutter_test.dart';
import 'package:theboringfloor/models/models.dart';

void main() {
  test('exec result decodes', () {
    final r = ExecResult.fromJson({
      'stdout': 'ok',
      'stderr': '',
      'exitCode': 0,
      'durationMs': 4,
      'truncated': false,
    });
    expect(r.stdout, 'ok');
  });
}
