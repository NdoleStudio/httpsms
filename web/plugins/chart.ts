import Vue from 'vue'
import { Bar } from 'vue-chartjs'
import {
  Chart as ChartJS,
  Title,
  Tooltip,
  Legend,
  BarElement,
  CategoryScale,
  LinearScale,
  TimeSeriesScale,
  LineElement,
  PointElement,
  ArcElement,
  TimeScale,
} from 'chart.js'

ChartJS.register(
  Title,
  Tooltip,
  Legend,
  PointElement,
  BarElement,
  TimeScale,
  TimeSeriesScale,
  CategoryScale,
  LinearScale,
  LineElement,
  ArcElement
)

Vue.component('BarChart', {
  extends: Bar,
})
