import 'package:flutter/material.dart';
import '../models/message.dart';

class ToolCallCard extends StatefulWidget {
  final ToolCall toolCall;
  final String? result;

  const ToolCallCard({
    super.key,
    required this.toolCall,
    this.result,
  });

  @override
  State<ToolCallCard> createState() => _ToolCallCardState();
}

class _ToolCallCardState extends State<ToolCallCard> {
  bool _expanded = false;

  IconData _getToolIcon(String name) {
    switch (name) {
      case 'bash':
        return Icons.terminal;
      case 'read_file':
        return Icons.description;
      case 'write_file':
        return Icons.edit_document;
      case 'edit_file':
        return Icons.edit;
      case 'search_files':
        return Icons.search;
      case 'grep_code':
        return Icons.manage_search;
      case 'webfetch':
        return Icons.web;
      case 'todo':
        return Icons.checklist;
      case 'save_memory':
        return Icons.book;
      case 'load_skill':
        return Icons.auto_stories;
      case 'delegate_subagent':
        return Icons.smart_toy;
      case 'task_create':
        return Icons.add_task;
      case 'task_update':
        return Icons.update;
      case 'task_list':
        return Icons.list;
      case 'task_get':
        return Icons.visibility;
      case 'background_run':
        return Icons.play_arrow;
      case 'check_background':
        return Icons.refresh;
      case 'cron_create':
        return Icons.schedule;
      case 'cron_delete':
        return Icons.delete_sweep;
      case 'cron_list':
        return Icons.list_alt;
      case 'compact':
        return Icons.compress;
      default:
        return Icons.build;
    }
  }

  Widget _buildResultWidget(String toolName, String result) {
    if (toolName == 'bash') {
      return _buildBashResult(result);
    }
    if (toolName == 'read_file') {
      return _buildReadFileResult(result);
    }
    if (toolName == 'todo') {
      return _buildTodoResult(result);
    }
    if (toolName == 'grep_code') {
      return _buildGrepResult(result);
    }
    if (toolName == 'background_run' || toolName == 'check_background') {
      return _buildBackgroundResult(result);
    }
    return _buildDefaultResult(result);
  }

  Widget _buildDefaultResult(String result) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(vertical: 8, horizontal: 12),
      decoration: BoxDecoration(
        color: Colors.grey[200],
        borderRadius: BorderRadius.circular(8),
      ),
      child: SelectableText(
        result,
        style: TextStyle(
          fontSize: 12,
          color: Colors.grey[800],
        ),
      ),
    );
  }

  Widget _buildBashResult(String result) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(vertical: 8, horizontal: 12),
      decoration: BoxDecoration(
        color: const Color(0xFF1E1E1E),
        borderRadius: BorderRadius.circular(8),
      ),
      child: SelectableText(
        result,
        style: const TextStyle(
          fontSize: 12,
          color: Color(0xFF4EC9B0),
          fontFamily: 'monospace',
        ),
      ),
    );
  }

  Widget _buildReadFileResult(String result) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(vertical: 8, horizontal: 12),
      decoration: BoxDecoration(
        color: const Color(0xFFF5F5F5),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: Colors.grey[300]!),
      ),
      child: SelectableText(
        result,
        style: TextStyle(
          fontSize: 12,
          color: Colors.grey[800],
          fontFamily: 'monospace',
        ),
      ),
    );
  }

  Widget _buildTodoResult(String result) {
    final lines = result.split('\n');
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: lines.map((line) {
        final isComplete = line.contains('[√]');
        final isProcessing = line.contains('[>]');
        return Padding(
          padding: const EdgeInsets.symmetric(vertical: 2),
          child: Row(
            children: [
              Icon(
                isComplete ? Icons.check_box : (isProcessing ? Icons.indeterminate_check_box : Icons.check_box_outline_blank),
                size: 14,
                color: isComplete ? Colors.green : (isProcessing ? Colors.orange : Colors.grey),
              ),
              const SizedBox(width: 4),
              Expanded(
                child: Text(
                  line.replaceAll('[√] ', '').replaceAll('[>] ', '').replaceAll('[ ] ', ''),
                  style: TextStyle(
                    fontSize: 12,
                    color: Colors.grey[800],
                  ),
                ),
              ),
            ],
          ),
        );
      }).toList(),
    );
  }

  Widget _buildGrepResult(String result) {
    final lines = result.split('\n');
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: lines.take(50).map((line) {
        final parts = line.split(':');
        if (parts.length >= 3) {
          return Container(
            padding: const EdgeInsets.symmetric(vertical: 1),
            child: RichText(
              text: TextSpan(
                children: [
                  TextSpan(
                    text: '${parts[0]}:${parts[1]}:',
                    style: TextStyle(
                      fontSize: 11,
                      color: Colors.grey[600],
                      fontFamily: 'monospace',
                    ),
                  ),
                  TextSpan(
                    text: parts.sublist(2).join(':'),
                    style: TextStyle(
                      fontSize: 11,
                      color: Colors.grey[800],
                      fontFamily: 'monospace',
                    ),
                  ),
                ],
              ),
            ),
          );
        }
        return Text(
          line,
          style: TextStyle(fontSize: 11, color: Colors.grey[800]),
        );
      }).toList(),
    );
  }

  Widget _buildBackgroundResult(String result) {
    final isRunning = result.contains('running');
    final isCompleted = result.contains('completed');
    final isError = result.contains('error') || result.contains('Error');
    final isTimeout = result.contains('timeout');

    Color statusColor;
    IconData statusIcon;
    String statusText;

    if (isRunning) {
      statusColor = Colors.blue;
      statusIcon = Icons.sync;
      statusText = 'Running';
    } else if (isCompleted) {
      statusColor = Colors.green;
      statusIcon = Icons.check_circle;
      statusText = 'Completed';
    } else if (isTimeout) {
      statusColor = Colors.orange;
      statusIcon = Icons.timer;
      statusText = 'Timeout';
    } else if (isError) {
      statusColor = Colors.red;
      statusIcon = Icons.error;
      statusText = 'Error';
    } else {
      statusColor = Colors.grey;
      statusIcon = Icons.info;
      statusText = 'Unknown';
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Icon(statusIcon, size: 14, color: statusColor),
            const SizedBox(width: 4),
            Text(
              statusText,
              style: TextStyle(
                fontSize: 12,
                fontWeight: FontWeight.w500,
                color: statusColor,
              ),
            ),
          ],
        ),
        const SizedBox(height: 4),
        Text(
          result,
          style: TextStyle(
            fontSize: 12,
            color: Colors.grey[800],
          ),
        ),
      ],
    );
  }

  @override
  Widget build(BuildContext context) {
    if (widget.toolCall.function.name == 'todo') {
      if (widget.result == null ||
          widget.result == 'no todo list' ||
          (widget.result?.contains('[Previous:') ?? false)) {
        return const SizedBox.shrink();
      }
      return _buildTodoOnly();
    }

    return Container(
      margin: const EdgeInsets.only(top: 8),
      decoration: BoxDecoration(
        color: Colors.grey[100],
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: Colors.grey[300]!),
      ),
      child: Column(
        children: [
          InkWell(
            onTap: () => setState(() => _expanded = !_expanded),
            borderRadius: BorderRadius.circular(8),
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
              child: Row(
                children: [
                  Icon(
                    _getToolIcon(widget.toolCall.function.name),
                    size: 16,
                    color: Colors.grey[700],
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(
                      widget.toolCall.function.name,
                      style: TextStyle(
                        fontWeight: FontWeight.w500,
                        color: Colors.grey[800],
                      ),
                    ),
                  ),
                  Icon(
                    _expanded ? Icons.expand_less : Icons.expand_more,
                    size: 18,
                    color: Colors.grey[600],
                  ),
                ],
              ),
            ),
          ),
          if (_expanded) ...[
            const Divider(height: 1),
            Padding(
              padding: const EdgeInsets.all(12),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    'Arguments:',
                    style: TextStyle(
                      fontWeight: FontWeight.w500,
                      fontSize: 12,
                      color: Colors.grey[600],
                    ),
                  ),
                  const SizedBox(height: 4),
                  Container(
                    width: double.infinity,
                    padding: const EdgeInsets.symmetric(vertical: 8, horizontal: 12),
                    decoration: BoxDecoration(
                      color: Colors.grey[200],
                      borderRadius: BorderRadius.circular(8),
                    ),
                    child: Text(
                      widget.toolCall.function.arguments,
                      style: TextStyle(
                        fontSize: 12,
                        color: Colors.grey[800],
                      ),
                    ),
                  ),
                  if (widget.result != null) ...[
                    const SizedBox(height: 12),
                    Text(
                      'Result:',
                      style: TextStyle(
                        fontWeight: FontWeight.w500,
                        fontSize: 12,
                        color: Colors.grey[600],
                      ),
                    ),
                    const SizedBox(height: 4),
                    _buildResultWidget(widget.toolCall.function.name, widget.result!),
                  ],
                ],
              ),
            ),
          ],
        ],
      ),
    );
  }

  Widget _buildTodoOnly() {
    return Container(
      margin: const EdgeInsets.only(top: 8),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: Colors.grey[100],
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: Colors.grey[300]!),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(
                _getToolIcon('todo'),
                size: 16,
                color: Colors.grey[700],
              ),
              const SizedBox(width: 8),
              Text(
                'Todo',
                style: TextStyle(
                  fontWeight: FontWeight.w500,
                  color: Colors.grey[800],
                ),
              ),
            ],
          ),
          if (widget.result != null) ...[
            const SizedBox(height: 8),
            _buildTodoResult(widget.result!),
          ],
        ],
      ),
    );
  }
}