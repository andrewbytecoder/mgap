import { BarChart, LineChart } from 'echarts/charts'
import { DatasetComponent, GridComponent, ToolboxComponent, TooltipComponent } from 'echarts/components'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'

use([BarChart, LineChart, DatasetComponent, GridComponent, ToolboxComponent, TooltipComponent, CanvasRenderer])
