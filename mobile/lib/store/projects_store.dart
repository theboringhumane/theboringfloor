import 'package:flutter/foundation.dart';

import '../api/gateway_client.dart';
import '../models/project.dart';

class ProjectsStore extends ChangeNotifier {
  ProjectsStore(this.client);
  final GatewayClient client;
  List<Project> projects = [];
  Object? error;
  bool loading = false;
  Future<void> load() async {
    loading = true;
    error = null;
    notifyListeners();
    try {
      projects = await client.projects();
    } catch (e) {
      error = e;
    }
    loading = false;
    notifyListeners();
  }
}
