import 'dart:async';

import 'package:flutter/widgets.dart';

class Polling extends StatefulWidget {
  const Polling({
    super.key,
    required this.interval,
    required this.onTick,
    required this.child,
    this.enabled = true,
    this.restartToken = 0,
    this.maxConsecutiveFailures = 3,
  });
  final Duration interval;

  /// Returns false when the request failed. Failures back off and eventually
  /// stop polling until [restartToken] changes.
  final Future<bool> Function() onTick;
  final Widget child;
  final bool enabled;
  final int restartToken;
  final int maxConsecutiveFailures;
  @override
  State<Polling> createState() => _PollingState();
}

class _PollingState extends State<Polling> with WidgetsBindingObserver {
  Timer? timer;
  bool _ticking = false;
  bool _foreground = true;
  int _consecutiveFailures = 0;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _schedule();
  }

  @override
  void didUpdateWidget(covariant Polling oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.restartToken != widget.restartToken) {
      _consecutiveFailures = 0;
    }
    if (oldWidget.enabled != widget.enabled ||
        oldWidget.interval != widget.interval ||
        oldWidget.restartToken != widget.restartToken) {
      _cancelAndSchedule();
    }
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    _foreground = state == AppLifecycleState.resumed;
    _cancelAndSchedule();
  }

  bool get _canPoll =>
      mounted &&
      widget.enabled &&
      _foreground &&
      _consecutiveFailures < widget.maxConsecutiveFailures;

  void _cancelAndSchedule() {
    timer?.cancel();
    timer = null;
    _schedule();
  }

  void _schedule() {
    if (!_canPoll || _ticking || timer != null) return;
    final multiplier = 1 << _consecutiveFailures.clamp(0, 3).toInt();
    timer = Timer(widget.interval * multiplier, () {
      timer = null;
      _tick();
    });
  }

  Future<void> _tick() async {
    if (_ticking || !_canPoll) return;

    _ticking = true;
    try {
      final succeeded = await widget.onTick();
      _consecutiveFailures = succeeded ? 0 : _consecutiveFailures + 1;
    } catch (_) {
      _consecutiveFailures += 1;
    } finally {
      _ticking = false;
      _schedule();
    }
  }

  @override
  void dispose() {
    timer?.cancel();
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => widget.child;
}
