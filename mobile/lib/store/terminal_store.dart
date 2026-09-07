import 'package:flutter/foundation.dart';

import '../api/gateway_client.dart';
import '../models/exec_result.dart';

class TerminalEntry {
  const TerminalEntry({required this.command, this.result, this.error});
  final String command;
  final ExecResult? result;
  final Object? error;
}

class TerminalStore extends ChangeNotifier {
  TerminalStore(this.client);
  final GatewayClient client;
  final List<TerminalEntry> entries = [];
  bool running = false;
  Future<void> run(String command, String cwd) async {
    if (running || command.trim().isEmpty) return;
    running = true;
    notifyListeners();
    try {
      entries.add(
        TerminalEntry(
          command: command,
          result: await client.exec(command: command, cwd: cwd),
        ),
      );
    } catch (e) {
      entries.add(TerminalEntry(command: command, error: e));
    }
    running = false;
    notifyListeners();
  }
}
