import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:theboringfloor/components/message_bubble.dart';
import 'package:theboringfloor/models/transcript.dart';

void main() {
  const message = TranscriptMessage(
    id: 'message-1',
    from: 'assistant',
    kind: 'text',
    text: '**Rendered** message',
    at: 0,
  );

  Widget subject(MessageKind kind) => MaterialApp(
    home: Scaffold(
      body: MessageBubble(
        message: message,
        expanded: false,
        onToggle: () {},
        kind: kind,
      ),
    ),
  );

  testWidgets('tool message carries its compact label', (tester) async {
    await tester.pumpWidget(subject(MessageKind.tool));

    expect(find.text('tool'), findsOneWidget);
    expect(find.byKey(const Key('tool-message')), findsOneWidget);
  });

  testWidgets('error message carries error presentation', (tester) async {
    await tester.pumpWidget(subject(MessageKind.error));

    expect(find.byKey(const Key('error-message')), findsOneWidget);
    expect(find.byIcon(Icons.error_outline), findsOneWidget);
  });
}
