import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:theboringfloor/components/message_bubble.dart';
import 'package:theboringfloor/components/transcript_attachment_chip.dart';
import 'package:theboringfloor/models/transcript.dart';
import 'package:theboringfloor/utils/typography.dart';

void main() {
  setUpAll(disableRuntimeFontFetching);

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

  testWidgets('shows attachment chips above the message body', (tester) async {
    const attachedMessage = TranscriptMessage(
      id: 'message-with-attachments',
      from: 'user',
      kind: 'text',
      text: 'Attached images',
      at: 0,
      attachments: [
        TranscriptAttachment(name: 'screenshot.png', mime: 'image/png'),
        TranscriptAttachment(
          name: 'a-very-long-screenshot-filename-that-must-ellipsis.png',
          mime: 'image/jpeg',
        ),
      ],
    );
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: MessageBubble(
            message: attachedMessage,
            expanded: false,
            onToggle: () {},
            kind: MessageKind.user,
          ),
        ),
      ),
    );

    expect(find.text('Attached images'), findsOneWidget);
    expect(find.text('PNG'), findsOneWidget);
    expect(find.text('JPEG'), findsOneWidget);
    expect(
      tester.widget<Text>(find.text('PNG').first).style?.fontFamily,
      'JetBrains Mono',
    );
    expect(find.byIcon(Icons.image_outlined), findsNWidgets(2));
  });

  testWidgets('without attachments retains no attachment chip', (tester) async {
    await tester.pumpWidget(subject(MessageKind.user));

    expect(find.byType(TranscriptAttachmentChip), findsNothing);
  });

  testWidgets('attachment chip omits a type label without MIME metadata', (
    tester,
  ) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(
          body: TranscriptAttachmentChip(
            attachment: TranscriptAttachment(name: 'untyped-image'),
          ),
        ),
      ),
    );

    expect(find.text('untyped-image'), findsOneWidget);
    expect(find.text('PNG'), findsNothing);
  });

  testWidgets('attachment chips wrap without overflow on a narrow screen', (
    tester,
  ) async {
    const attachedMessage = TranscriptMessage(
      id: 'narrow-message',
      from: 'user',
      kind: 'text',
      text: 'Images',
      at: 0,
      attachments: [
        TranscriptAttachment(name: 'one.png', mime: 'image/png'),
        TranscriptAttachment(name: 'two.jpeg', mime: 'image/jpeg'),
        TranscriptAttachment(name: 'three.png', mime: 'image/png'),
        TranscriptAttachment(
          name: 'a-very-long-screenshot-filename-that-must-not-overflow.png',
          mime: 'image/png',
        ),
      ],
    );
    await tester.binding.setSurfaceSize(const Size(320, 640));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: MessageBubble(
            message: attachedMessage,
            expanded: false,
            onToggle: () {},
            kind: MessageKind.user,
          ),
        ),
      ),
    );

    expect(tester.takeException(), isNull);
    expect(find.byType(TranscriptAttachmentChip), findsNWidgets(4));
  });
}
