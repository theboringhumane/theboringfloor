import 'package:flutter/foundation.dart';

import '../api/gateway_client.dart';
import '../models/floor.dart';
import '../models/project.dart';
import '../models/session.dart';

class FloorPulse {
  const FloorPulse({
    required this.project,
    this.status,
    this.floor,
    this.problem,
  });
  final Project project;
  final Status? status;
  final FloorData? floor;
  final String? problem;
  List<FloorTicket> get review =>
      floor?.tickets.where((t) => t.status == 'review').toList() ?? [];
  List<FloorTicket> get blocked =>
      floor?.tickets.where((t) => t.status == 'blocked').toList() ?? [];
  bool get needsAttention =>
      problem != null ||
      status?.execution?.needsAttention == true ||
      (status?.execution == null && status?.planPending == true) ||
      review.isNotEmpty ||
      blocked.isNotEmpty;
  bool get working =>
      {'working', 'planning'}.contains(status?.execution?.state);
}

/// Four workers bound remote requests even when a gateway has many floors.
/// A failed refresh retains the previous snapshot and labels it as stale.
class AttentionStore extends ChangeNotifier {
  AttentionStore(this.client);
  final GatewayClient client;
  List<FloorPulse> floors = [];
  bool loading = false;
  String? error;
  DateTime? updatedAt;
  bool _disposed = false;
  int get needsAttention => floors.where((p) => p.needsAttention).length;
  int get working => floors.where((p) => p.working).length;
  Future<bool> refresh() async {
    if (loading || _disposed) return true;
    loading = true;
    notifyListeners();
    try {
      final projects = await client.projects();
      final previous = {for (final p in floors) p.project.id: p};
      final next = List<FloorPulse?>.filled(projects.length, null);
      var cursor = 0;
      Future<void> worker() async {
        while (!_disposed && cursor < projects.length) {
          final index = cursor++;
          final project = projects[index];
          Status? status;
          FloorData? floor;
          String? problem;
          try {
            floor = await client.floor(project.id);
          } catch (_) {
            floor = previous[project.id]?.floor;
            problem =
                'Board unavailable. Showing saved information if available.';
          }
          if (project.live) {
            try {
              status = await client.status(project.id);
            } catch (_) {
              problem =
                  'Live status unavailable. Open the floor or try refreshing.';
            }
          }
          next[index] = FloorPulse(
            project: project,
            status: status,
            floor: floor,
            problem: problem,
          );
        }
      }

      await Future.wait(
        List.generate(projects.length.clamp(0, 4), (_) => worker()),
      );
      if (_disposed) return false;
      floors = next.whereType<FloorPulse>().toList()
        ..sort((a, b) {
          final priority =
              (b.needsAttention
                  ? 2
                  : b.working
                  ? 1
                  : 0) -
              (a.needsAttention
                  ? 2
                  : a.working
                  ? 1
                  : 0);
          return priority != 0
              ? priority
              : a.project.name.compareTo(b.project.name);
        });
      updatedAt = DateTime.now();
      error = null;
      return true;
    } catch (_) {
      if (!_disposed) error = 'Could not refresh your floors. Check the gateway connection. The view may be out of date.';
      return false;
    } finally {
      loading = false;
      if (!_disposed) notifyListeners();
    }
  }

  @override
  void dispose() {
    _disposed = true;
    super.dispose();
  }
}
