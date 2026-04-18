import 'package:flutter/foundation.dart';
import '../models/session.dart';
import '../models/project.dart';
import '../models/event.dart';
import '../services/api.dart';

class SessionProvider extends ChangeNotifier {
  final ApiService _api = ApiService();
  
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
  
  // ============ Project Methods ============
  
  Future<void> loadProjects() async {
    _isLoadingProjects = true;
    _error = null;
    notifyListeners();
    
    try {
      _projects = await _api.listProjects();
      _isLoadingProjects = false;
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
      notifyListeners();
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
  
  void _updateSession(Session session) {
    final index = _sessions.indexWhere((s) => s.id == session.id);
    if (index != -1) {
      _sessions[index] = session;
    }
  }
  
  void handleEvent(SessionEvent event) {
    final index = _sessions.indexWhere((s) => s.id == event.sessionId);
    if (index == -1) return;
    
    final session = _sessions[index];
    SessionState? newState;
    String? blockedOn = session.blockedOn;
    String? blockedTool = session.blockedTool;
    Map<String, dynamic>? blockedArgs = session.blockedArgs;
    
    switch (event.type) {
      case 'state_change':
        final stateStr = event.data['new_state'] as String? ?? event.data['state'] as String?;
        if (stateStr != null) {
          newState = SessionState.values.firstWhere(
            (e) => e.name == stateStr,
            orElse: () => session.state,
          );
          if (event.data['unblocked'] == true) {
            blockedOn = null;
            blockedTool = null;
            blockedArgs = null;
          }
        }
        break;
      case 'blocked':
        newState = SessionState.blocked;
        blockedOn = event.data['blocked_on'] as String?;
        blockedTool = event.data['blocked_tool'] as String?;
        blockedArgs = (event.data['blocked_args'] as Map?)?.cast<String, dynamic>();
        break;
      case 'completed':
        newState = SessionState.completed;
        blockedOn = null;
        blockedTool = null;
        blockedArgs = null;
        break;
      default:
        break;
    }
    
    if (newState != null && newState != session.state ||
        blockedOn != session.blockedOn ||
        blockedTool != session.blockedTool ||
        blockedArgs != session.blockedArgs) {
      _sessions[index] = Session(
        id: session.id,
        projectPath: session.projectPath,
        model: session.model,
        state: newState ?? session.state,
        mode: session.mode,
        createdAt: session.createdAt,
        input: session.input,
        messages: session.messages,
        blockedOn: blockedOn,
        blockedTool: blockedTool,
        blockedArgs: blockedArgs,
      );
      notifyListeners();
    }
  }
  
  void clearError() {
    _error = null;
    notifyListeners();
  }
}