import 'package:flutter_test/flutter_test.dart';
import 'package:theboringfloor/utils/time.dart';

void main() {
  test(
    'formats time',
    () => expect(
      formatRelativeTime(DateTime(2026), now: DateTime(2026, 1, 1, 0, 1)),
      '1m ago',
    ),
  );
}
