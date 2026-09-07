import 'dart:async';

import 'package:flutter/widgets.dart';

class Polling extends StatefulWidget {
  const Polling({
    super.key,
    required this.interval,
    required this.onTick,
    required this.child,
  });
  final Duration interval;
  final Future<void> Function() onTick;
  final Widget child;
  @override
  State<Polling> createState() => _PollingState();
}

class _PollingState extends State<Polling> {
  Timer? timer;
  bool _ticking = false;

  @override
  void initState() {
    super.initState();
    timer = Timer.periodic(widget.interval, (_) => _tick());
  }

  Future<void> _tick() async {
    if (_ticking) return;

    _ticking = true;
    try {
      await widget.onTick();
    } finally {
      _ticking = false;
    }
  }

  @override
  void dispose() {
    timer?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => widget.child;
}
