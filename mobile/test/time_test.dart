import 'package:flutter_test/flutter_test.dart';
import 'package:theboringfloor/time.dart';

void main() {
  final now = DateTime(2026, 9, 7, 12);
  test(
    'formats recent time',
    () => expect(
      formatRelativeTime(now.subtract(const Duration(seconds: 9)), now: now),
      'just now',
    ),
  );
  test('formats minutes and days', () {
    expect(
      formatRelativeTime(now.subtract(const Duration(minutes: 4)), now: now),
      '4m ago',
    );
    expect(
      formatRelativeTime(now.subtract(const Duration(days: 2)), now: now),
      '2d ago',
    );
  });
}
