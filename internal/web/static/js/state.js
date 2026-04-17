const API = '';
let allGoals = [];
let allActs = [];
let currentGoalID = null;
let pendingGoalID = null;

// ECharts instances — disposed & recreated on each dashboard load
let chartVolume, chartSportDist, chartPace, chartHR;
