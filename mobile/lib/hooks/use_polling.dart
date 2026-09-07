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
  @override
  void initState() {
    super.initState();
    timer = Timer.periodic(widget.interval, (_) => widget.onTick());
  }

  @override
  void dispose() {
    timer?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => widget.child;
}
