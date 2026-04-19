import 'dart:async';
import 'package:flutter/foundation.dart';
import '../models/session.dart';
import '../models/project.dart';
import '../models/event.dart';
import '../services/api.dart';
import '../services/sse.dart';

class SessionProvider extends ChangeNotifier {
  final ApiService _api = ApiService();
  final SseService _sseService = SseService();
  final Map<String, StreamSubscription<SessionEvent>> _sseSubscriptions = {};
  Timer? _refreshTimer;

  List<Project> _projects = [];
  List<Session> _sessions = [];
  Project? _selectedProject;
  bool _isLoading = false;
  bool _isLoadingProjects = false;
  String? _error;

  List<Project> get projects => _projects;
  List<Session> get sessions => _sessions;
  Project? get selectedProject => _selectedProject;
  bool get isLoading => _isLoading;
  bool get isLoadingProjects => _isLoadingProjects;
  String? get error => _error;

  List<Session> get pendingSessions => _sessions.where((s) => s.state == SessionState.pending).toList();
  List<Session> get planningSessions => _sessions.where((s) => s.state == SessionState.planning).toList();
  List<Session> get processingSessions => _sessions.where((s) => s.state == SessionState.processing).toList();
  List<Session> get blockedSessions => _sessions.where((s) => s.state == SessionState.blocked).toList();
  List<Session> get completedSessions => _sessions.where((s) => s.state == SessionState.completed).toList();

  @override
  void dispose() {
    _refreshTimer?.cancel();
    for (final subscription in _sseSubscriptions.values) {
      subscription.cancel();
    }
    _sseSubscriptions.clear();
    super.dispose();
  }

  void subscribeToSession(String sessionId) {
    if (_sseSubscriptions.containsKey(sessionId)) return;

    final stream = _sseService.connect(sessionId);
    final subscription = stream.listen(
      (event) => handleEvent(event),
      onError: (error) {
        _error = 'SSE error: $error';
        notifyListeners();
      },
    );
    _sseSubscriptions[sessionId] = subscription;
  }

  void unsubscribeFromSession(String sessionId) {
    final subscription = _sseSubscriptions.remove(sessionId);
    subscription?.cancel();
  }

  Future<void> _fetchSessionMessages(String sessionId) async {
    try {
      final session = await _api.getSession(sessionId);
      _updateSession(session);
      notifyListeners();
    } catch (e) {
      _error = e.toString();
      notifyListeners();
    }
  }

  void _refreshSessionsDebounced() {
    _refreshTimer?.cancel();
    _refreshTimer = Timer(const Duration(milliseconds: 300), () {
      if (_selectedProject != null) {
        loadSessions(projectName: _selectedProject!.name);
      }
    });
  }

  // ============ Project Methods ============

  Future<void> loadProjects() async {
    _isLoadingProjects = true;
    _error = null;
    notifyListeners();

    try {
      _projects = await _api.listProjects();
      _isLoadingProjects = false;

      if (_selectedProject == null && _projects.isNotEmpty) {
        _selectedProject = _projects.first;
        loadSessions(projectName: _selectedProject!.name);
      }

      notifyListeners();
    } catch (e) {
      _error = e.toString();
      _isLoadingProjects = false;
      notifyListeners();
    }
  }

  Future<void> createProject(String path) async {
    try {
      await _api.createProject(path);
      await loadProjects();
    } catch (e) {
      _error = e.toString();
      notifyListeners();
    }
  }

  void selectProject(Project? project) {
    _selectedProject = project;
    if (project != null) {
      loadSessions(projectName: project.name);
    } else {
      loadSessions();
    }
    notifyListeners();
  }

  // ============ Session Methods ============

  Future<void> loadSessions({String? projectName}) async {
    _isLoading = true;
    _error = null;
    notifyListeners();

    try {
      _sessions = await _api.listSessions(projectName: projectName);
      _isLoading = false;
      notifyListeners();
    } catch (e) {
      _error = e.toString();
      _isLoading = false;
      notifyListeners();
    }
  }

  Future<Session> createSession(String projectPath, {String? model}) async {
    try {
      final session = await _api.createSession(projectPath, model: model);
      _sessions.add(session);
      notifyListeners();
      return session;
    } catch (e) {
      _error = e.toString();
      notifyListeners();
      rethrow;
    }
  }

  Future<void> submitInput(String sessionId, String input, {String? mode}) async {
    try {
      final session = await _api.submitInput(sessionId, input, mode: mode);
      _updateSession(session);
      subscribeToSession(sessionId);
      await _fetchSessionMessages(sessionId);
    } catch (e) {
      _error = e.toString();
      notifyListeners();
    }
  }

  Future<void> approve(String sessionId) async {
    try {
      final session = await _api.approve(sessionId);
      _updateSession(session);
      notifyListeners();
    } catch (e) {
      _error = e.toString();
      notifyListeners();
    }
  }

  Future<void> unblock(String sessionId, bool approved, {String? addAllowed}) async {
    try {
      final session = await _api.unblock(sessionId, approved, addAllowed: addAllowed);
      _updateSession(session);
      notifyListeners();
    } catch (e) {
      _error = e.toString();
      notifyListeners();
    }
  }

  Future<void> removeSession(String sessionId) async {
    try {
      final matchingSessions = _sessions.where((s) => s.id == sessionId).toList();
      if (matchingSessions.isEmpty) return;

      final session = matchingSessions.first;
      await _api.removeSessionFromProject(session.projectName, sessionId);
      _sessions.removeWhere((s) => s.id == sessionId);
      unsubscribeFromSession(sessionId);
      notifyListeners();
    } catch (e) {
      _error = e.toString();
      notifyListeners();
    }
  }

  void _updateSession(Session session) {
    final index = _sessions.indexWhere((s) => s.id == session.id);
    if (index != -1) {
      _sessions[index] = session;
    }
  }

  void handleEvent(SessionEvent event) {
    _fetchSessionMessages(event.sessionId);

    switch (event.type) {
      case 'state_change':
        _refreshSessionsDebounced();
        break;
      case 'blocked':
        _refreshSessionsDebounced();
        break;
      case 'completed':
        unsubscribeFromSession(event.sessionId);
        break;
    }
  }

  void clearError() {
    _error = null;
    notifyListeners();
  }
}