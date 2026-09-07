import 'package:flutter/material.dart';

import '../api/gateway_client.dart';
import '../store/terminal_store.dart';
import '../utils/typography.dart';

class TerminalView extends StatefulWidget {
  const TerminalView({super.key, required this.store, required this.cwd});
  final TerminalStore store;
  final String cwd;
  @override
  State<TerminalView> createState() => _TerminalViewState();
}

class _TerminalViewState extends State<TerminalView> {
  final command = TextEditingController();
  @override
  void initState() {
    super.initState();
    widget.store.addListener(_changed);
  }

  @override
  void dispose() {
    widget.store.removeListener(_changed);
    command.dispose();
    super.dispose();
  }

  void _changed() {
    if (mounted) setState(() {});
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final codeStyle = AppFonts.mono(context, base: theme.textTheme.bodyMedium);
    return SafeArea(
      child: Column(
        children: [
          Padding(
            padding: const EdgeInsets.all(20),
            child: Text(
              'Terminal',
              style: AppFonts.heading(
                context,
                base: theme.textTheme.headlineLarge,
              ),
            ),
          ),
          Expanded(
            child: ListView(
              children: widget.store.entries.map((entry) {
                final result = entry.result;
                final disabled =
                    entry.error is GatewayException &&
                    (entry.error as GatewayException).statusCode == 403;
                return Padding(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 20,
                    vertical: 12,
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(r'$ ${entry.command}', style: codeStyle),
                      if (disabled)
                        const Text('Remote exec is disabled on the server.'),
                      if (entry.error != null && !disabled)
                        Text(
                          '${entry.error}',
                          style: theme.textTheme.bodyMedium?.copyWith(
                            color: theme.colorScheme.error,
                          ),
                        ),
                      if (result != null) ...[
                        Text(
                          'exit ${result.exitCode} · ${result.durationMs}ms',
                          style: AppFonts.mono(
                            context,
                            base: theme.textTheme.bodyMedium?.copyWith(
                              color: result.exitCode == 0
                                  ? theme.colorScheme.primary
                                  : theme.colorScheme.error,
                            ),
                          ),
                        ),
                        if (result.stdout.isNotEmpty)
                          Text(result.stdout, style: codeStyle),
                        if (result.stderr.isNotEmpty)
                          Text(
                            result.stderr,
                            style: codeStyle.copyWith(
                              color: theme.colorScheme.error,
                            ),
                          ),
                        if (result.truncated)
                          Text('Output truncated', style: codeStyle),
                      ],
                    ],
                  ),
                );
              }).toList(),
            ),
          ),
          Padding(
            padding: const EdgeInsets.all(16),
            child: Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: command,
                    enabled: !widget.store.running,
                    style: AppFonts.mono(
                      context,
                      base: theme.textTheme.bodyLarge,
                    ),
                    onSubmitted: (_) => _run(),
                    decoration: const InputDecoration(
                      hintText: 'Enter command',
                    ),
                  ),
                ),
                IconButton(
                  tooltip: 'Run command',
                  onPressed: widget.store.running ? null : _run,
                  icon: widget.store.running
                      ? const SizedBox(
                          width: 18,
                          height: 18,
                          child: CircularProgressIndicator(),
                        )
                      : const Icon(Icons.play_arrow),
                ),
              ],
            ),
          ),
          if (widget.store.running)
            Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: Text('Running…', style: codeStyle),
            ),
        ],
      ),
    );
  }

  void _run() {
    final text = command.text;
    command.clear();
    widget.store.run(text, widget.cwd);
  }
}
